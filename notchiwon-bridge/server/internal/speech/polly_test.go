package speech

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/config"
)

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
	cfg := config.Polly{VoiceID: "Seoyeon", Engine: "neural", Rate: "90%"}
	api := &fakePolly{audio: "mp3"}
	audio, err := NewPolly(cfg, api).Synthesize(context.Background(), "안녕하세요 <김> & 박", "nara")
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if string(audio) != "mp3" {
		t.Errorf("audio = %q", audio)
	}
	in := api.got
	// The Clova speaker name is ignored: Polly speaks with POLLY_VOICE.
	if in.VoiceId != types.VoiceIdSeoyeon || in.Engine != types.EngineNeural || in.OutputFormat != types.OutputFormatMp3 || in.LanguageCode != types.LanguageCodeKoKr {
		t.Errorf("input = %+v", in)
	}
	want := `<speak><prosody rate="90%">안녕하세요 &lt;김&gt; &amp; 박</prosody></speak>`
	if in.TextType != types.TextTypeSsml || *in.Text != want {
		t.Errorf("text = %s %q, want ssml %q", in.TextType, *in.Text, want)
	}
}

func TestPollyPlainText(t *testing.T) {
	api := &fakePolly{audio: "mp3"}
	if _, err := NewPolly(config.Polly{VoiceID: "Jihye", Engine: "neural", Rate: "100%"}, api).Synthesize(context.Background(), "네", "default"); err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	if api.got.TextType != types.TextTypeText || *api.got.Text != "네" {
		t.Errorf("text = %s %q", api.got.TextType, *api.got.Text)
	}
}

func TestPollyErrors(t *testing.T) {
	cfg := config.Polly{VoiceID: "Seoyeon", Engine: "neural"}
	if _, err := NewPolly(cfg, &fakePolly{err: errors.New("denied")}).Synthesize(context.Background(), "네", ""); err == nil || !strings.Contains(err.Error(), "denied") {
		t.Errorf("err = %v, want the API error", err)
	}
	if _, err := NewPolly(cfg, &fakePolly{}).Synthesize(context.Background(), "네", ""); err == nil {
		t.Error("empty audio: want an error")
	}
}

func TestNewSynthesizer(t *testing.T) {
	ctx := context.Background()
	s, err := NewSynthesizer(ctx, config.Speech{Voice: config.Voice{Provider: config.TTSClova}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(Unavailable); !ok {
		t.Errorf("clova without keys = %T, want Unavailable", s)
	}
	s, err = NewSynthesizer(ctx, config.Speech{
		Clova: config.Clova{ClientID: "id", ClientSecret: "secret"},
		Voice: config.Voice{Provider: config.TTSClova},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*Clova); !ok {
		t.Errorf("clova with keys = %T, want *Clova", s)
	}
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	s, err = NewSynthesizer(ctx, config.Speech{
		Clova: config.Clova{ClientID: "id", ClientSecret: "secret"},
		Voice: config.Voice{Provider: config.TTSPolly, Polly: config.Polly{Region: "ap-northeast-2", VoiceID: "Seoyeon", Engine: "neural"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*Polly); !ok {
		t.Errorf("polly = %T, want *Polly", s)
	}
}
