package speech

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	ptypes "github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
	ttypes "github.com/aws/aws-sdk-go-v2/service/transcribestreaming/types"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/config"
)

// fakeWriter records the audio chunks sent and, on Close, plays the
// results the fake service returns.
type fakeWriter struct {
	mu      sync.Mutex
	chunks  [][]byte
	closed  bool
	sendErr error
	onClose func()
}

func (w *fakeWriter) Send(_ context.Context, ev ttypes.AudioStream) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.sendErr != nil {
		return w.sendErr
	}
	w.chunks = append(w.chunks, append([]byte(nil), ev.(*ttypes.AudioStreamMemberAudioEvent).Value.AudioChunk...))
	return nil
}

func (w *fakeWriter) Close() error {
	w.mu.Lock()
	already := w.closed
	w.closed = true
	w.mu.Unlock()
	if !already && w.onClose != nil {
		w.onClose()
	}
	return nil
}

func (w *fakeWriter) Err() error { return nil }

type fakeReader struct {
	events chan ttypes.TranscriptResultStream
	err    error
	once   sync.Once
}

func (r *fakeReader) Events() <-chan ttypes.TranscriptResultStream { return r.events }
func (r *fakeReader) Close() error {
	r.once.Do(func() { close(r.events) })
	return nil
}
func (r *fakeReader) Err() error { return r.err }

func result(text string, partial bool) ttypes.Result {
	return ttypes.Result{IsPartial: partial, Alternatives: []ttypes.Alternative{{Transcript: aws.String(text)}}}
}

// fakeTranscribe answers every stream with results once the input is closed.
type fakeTranscribe struct {
	in      *transcribestreaming.StartStreamTranscriptionInput
	writer  *fakeWriter
	results []ttypes.Result
	readErr error
	sendErr error
}

func (f *fakeTranscribe) start(_ context.Context, in *transcribestreaming.StartStreamTranscriptionInput) (*transcribestreaming.StartStreamTranscriptionEventStream, error) {
	f.in = in
	r := &fakeReader{events: make(chan ttypes.TranscriptResultStream, 8), err: f.readErr}
	f.writer = &fakeWriter{sendErr: f.sendErr, onClose: func() {
		for _, res := range f.results {
			r.events <- &ttypes.TranscriptResultStreamMemberTranscriptEvent{Value: ttypes.TranscriptEvent{
				Transcript: &ttypes.Transcript{Results: []ttypes.Result{res}},
			}}
		}
		_ = r.Close()
	}}
	return transcribestreaming.NewStartStreamTranscriptionEventStream(func(es *transcribestreaming.StartStreamTranscriptionEventStream) {
		es.Writer = f.writer
		es.Reader = r
	}), nil
}

func TestTranscribe(t *testing.T) {
	f := &fakeTranscribe{results: []ttypes.Result{
		result("손주가", true),
		result(" 손주가 어제 왔어 ", false),
		result("그래서 좋았어", false),
	}}
	pcm := make([]byte, 7000) // 3200-byte (100 ms) chunks: 3200 + 3200 + 600
	text, err := NewTranscribe("ko-KR", f.start).Recognize(context.Background(), pcm, "pcm16", 16000)
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}
	if text != "손주가 어제 왔어 그래서 좋았어" {
		t.Errorf("text = %q (partial results must be skipped)", text)
	}
	in := f.in
	if in.LanguageCode != ttypes.LanguageCodeKoKr || in.MediaEncoding != ttypes.MediaEncodingPcm || *in.MediaSampleRateHertz != 16000 {
		t.Errorf("input = %+v", in)
	}
	if n := len(f.writer.chunks); n != 3 || len(f.writer.chunks[0]) != 3200 || len(f.writer.chunks[2]) != 600 || !f.writer.closed {
		t.Errorf("chunks = %d, closed = %v", n, f.writer.closed)
	}
}

func TestTranscribeFormats(t *testing.T) {
	f := &fakeTranscribe{results: []ttypes.Result{result("네", false)}}
	tr := NewTranscribe("ko-KR", f.start)
	if _, err := tr.Recognize(context.Background(), []byte("OggS"), "opus", 0); err != nil {
		t.Fatalf("opus: %v", err)
	}
	if f.in.MediaEncoding != ttypes.MediaEncodingOggOpus || *f.in.MediaSampleRateHertz != 48000 {
		t.Errorf("opus input = %+v", f.in)
	}
	if _, err := tr.Recognize(context.Background(), []byte{1, 2}, "aac", 0); !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf("aac err = %v, want ErrUnsupportedFormat", err)
	}
}

func TestTranscribeErrors(t *testing.T) {
	ctx := context.Background()
	f := &fakeTranscribe{sendErr: errors.New("broken pipe")}
	if _, err := NewTranscribe("ko-KR", f.start).Recognize(ctx, make([]byte, 10), "pcm16", 16000); err == nil || !strings.Contains(err.Error(), "broken pipe") {
		t.Errorf("send err = %v", err)
	}
	f = &fakeTranscribe{readErr: errors.New("BadRequestException")}
	if _, err := NewTranscribe("ko-KR", f.start).Recognize(ctx, make([]byte, 10), "pcm16", 16000); err == nil || !strings.Contains(err.Error(), "BadRequestException") {
		t.Errorf("stream err = %v", err)
	}
	failing := func(context.Context, *transcribestreaming.StartStreamTranscriptionInput) (*transcribestreaming.StartStreamTranscriptionEventStream, error) {
		return nil, errors.New("AccessDenied")
	}
	if _, err := NewTranscribe("ko-KR", failing).Recognize(ctx, make([]byte, 10), "pcm16", 16000); err == nil || !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("start err = %v", err)
	}
}

type fakePolly struct {
	got   *polly.SynthesizeSpeechInput
	audio string
	err   error
}

func (f *fakePolly) SynthesizeSpeech(_ context.Context, in *polly.SynthesizeSpeechInput, _ ...func(*polly.Options)) (*polly.SynthesizeSpeechOutput, error) {
	f.got = in
	if f.err != nil {
		return nil, f.err
	}
	return &polly.SynthesizeSpeechOutput{AudioStream: io.NopCloser(strings.NewReader(f.audio))}, nil
}

func TestPolly(t *testing.T) {
	api := &fakePolly{audio: "ID3mp3"}
	audio, err := NewPolly("Seoyeon", "neural", "90%", api).Synthesize(context.Background(), "안녕하세요 <김> & 박", "default")
	if err != nil || string(audio) != "ID3mp3" {
		t.Fatalf("Synthesize = %q, %v", audio, err)
	}
	in := api.got
	if in.VoiceId != ptypes.VoiceIdSeoyeon || in.Engine != ptypes.EngineNeural || in.OutputFormat != ptypes.OutputFormatMp3 || in.LanguageCode != ptypes.LanguageCodeKoKr {
		t.Errorf("input = %+v", in)
	}
	want := `<speak><prosody rate="90%">안녕하세요 &lt;김&gt; &amp; 박</prosody></speak>`
	if in.TextType != ptypes.TextTypeSsml || *in.Text != want {
		t.Errorf("text = %s %q, want ssml %q", in.TextType, *in.Text, want)
	}
}

func TestPollyVoiceAndRate(t *testing.T) {
	ctx := context.Background()
	api := &fakePolly{audio: "mp3"}
	p := NewPolly("Seoyeon", "neural", "100%", api)
	if _, err := p.Synthesize(ctx, "네", "Jihye"); err != nil {
		t.Fatal(err)
	}
	if api.got.VoiceId != ptypes.VoiceIdJihye || api.got.TextType != ptypes.TextTypeText || *api.got.Text != "네" {
		t.Errorf("Jihye plain = %+v", api.got)
	}
	// An unknown voice (say a leftover speaker name) falls back to POLLY_VOICE.
	if _, err := p.Synthesize(ctx, "네", "nara"); err != nil {
		t.Fatal(err)
	}
	if api.got.VoiceId != ptypes.VoiceIdSeoyeon {
		t.Errorf("unknown voice = %s, want Seoyeon", api.got.VoiceId)
	}
}

func TestPollyErrors(t *testing.T) {
	ctx := context.Background()
	if _, err := NewPolly("Seoyeon", "neural", "", &fakePolly{err: errors.New("denied")}).Synthesize(ctx, "네", ""); err == nil || !strings.Contains(err.Error(), "denied") {
		t.Errorf("err = %v", err)
	}
	if _, err := NewPolly("Seoyeon", "neural", "", &fakePolly{}).Synthesize(ctx, "네", ""); err == nil {
		t.Error("empty audio: want an error")
	}
}

func TestNew(t *testing.T) {
	ctx := context.Background()
	stt, tts, err := New(ctx, config.Speech{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := stt.(Unavailable); !ok {
		t.Errorf("stt without keys = %T", stt)
	}
	if _, err := tts.Synthesize(ctx, "네", ""); !errors.Is(err, ErrUnavailable) {
		t.Errorf("tts without keys err = %v", err)
	}
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	stt, tts, err = New(ctx, config.Speech{AccessKeyID: "AKIAEXAMPLE", SecretAccessKey: "secret", Region: "ap-southeast-2", LanguageCode: "ko-KR", Voice: "Seoyeon", Engine: "neural"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := stt.(*Transcribe); !ok {
		t.Errorf("stt = %T, want *Transcribe", stt)
	}
	if _, ok := tts.(*Polly); !ok {
		t.Errorf("tts = %T, want *Polly", tts)
	}
}

func TestTranscribeTimeout(t *testing.T) {
	// The service never answers: Recognize must give up when ctx ends.
	hang := func(context.Context, *transcribestreaming.StartStreamTranscriptionInput) (*transcribestreaming.StartStreamTranscriptionEventStream, error) {
		return transcribestreaming.NewStartStreamTranscriptionEventStream(func(es *transcribestreaming.StartStreamTranscriptionEventStream) {
			es.Writer = &fakeWriter{}
			es.Reader = &fakeReader{events: make(chan ttypes.TranscriptResultStream)}
		}), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := NewTranscribe("ko-KR", hang).Recognize(ctx, make([]byte, 10), "pcm16", 16000); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
}
