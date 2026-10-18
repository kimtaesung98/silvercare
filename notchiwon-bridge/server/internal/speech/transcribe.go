package speech

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming/types"
)

// ErrUnsupportedFormat is an elder.audio format Transcribe streaming cannot
// read (it takes PCM, Ogg Opus and FLAC).
var ErrUnsupportedFormat = errors.New("audio format not supported by Amazon Transcribe")

// chunkMillis is how much audio goes in one stream event. AWS recommends
// 50-200 ms chunks.
const chunkMillis = 100

// Starter opens a Transcribe streaming session (a fake in tests).
type Starter func(ctx context.Context, in *transcribestreaming.StartStreamTranscriptionInput) (*transcribestreaming.StartStreamTranscriptionEventStream, error)

// StreamStarter adapts the SDK client to a Starter.
func StreamStarter(c *transcribestreaming.Client) Starter {
	return func(ctx context.Context, in *transcribestreaming.StartStreamTranscriptionInput) (*transcribestreaming.StartStreamTranscriptionEventStream, error) {
		out, err := c.StartStreamTranscription(ctx, in)
		if err != nil {
			return nil, err
		}
		return out.GetStream(), nil
	}
}

// Transcribe recognizes one finished utterance with Amazon Transcribe
// streaming: it opens a stream, sends the whole recording in chunks, closes
// the input and joins the final (non-partial) results.
type Transcribe struct {
	language types.LanguageCode
	start    Starter
}

// NewTranscribe returns a Transcribe for language (e.g. ko-KR).
func NewTranscribe(language string, start Starter) *Transcribe {
	return &Transcribe{language: types.LanguageCode(language), start: start}
}

// Recognize implements Recognizer.
func (t *Transcribe) Recognize(ctx context.Context, audio []byte, format string, sampleRate int) (string, error) {
	in := &transcribestreaming.StartStreamTranscriptionInput{LanguageCode: t.language}
	chunk := 4096
	switch format {
	case "pcm16", "":
		if sampleRate == 0 {
			sampleRate = 16000
		}
		in.MediaEncoding = types.MediaEncodingPcm
		chunk = sampleRate * 2 * chunkMillis / 1000 // 16-bit mono
	case "opus":
		if sampleRate == 0 {
			sampleRate = 48000
		}
		in.MediaEncoding = types.MediaEncodingOggOpus
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
	in.MediaSampleRateHertz = aws.Int32(int32(sampleRate))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := t.start(ctx, in)
	if err != nil {
		return "", fmt.Errorf("transcribe: %w", err)
	}
	defer stream.Close()

	sent := make(chan error, 1)
	go func() {
		err := send(ctx, stream, audio, chunk)
		sent <- err
		if err != nil {
			_ = stream.Close() // ends Events() so the loop below returns
		}
	}()

	var parts []string
	events := stream.Events()
read:
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case ev, ok := <-events:
			if !ok {
				break read
			}
			te, isTranscript := ev.(*types.TranscriptResultStreamMemberTranscriptEvent)
			if !isTranscript || te.Value.Transcript == nil {
				continue
			}
			for _, r := range te.Value.Transcript.Results {
				if r.IsPartial || len(r.Alternatives) == 0 || r.Alternatives[0].Transcript == nil {
					continue
				}
				parts = append(parts, strings.TrimSpace(*r.Alternatives[0].Transcript))
			}
		}
	}
	if err := <-sent; err != nil {
		return "", fmt.Errorf("transcribe send: %w", err)
	}
	if err := stream.Err(); err != nil {
		return "", fmt.Errorf("transcribe: %w", err)
	}
	return strings.TrimSpace(strings.Join(parts, " ")), nil
}

// send writes audio in chunk-sized events, then closes the input so
// Transcribe finishes and closes the result stream.
func send(ctx context.Context, stream *transcribestreaming.StartStreamTranscriptionEventStream, audio []byte, chunk int) error {
	for len(audio) > 0 {
		n := min(chunk, len(audio))
		if err := stream.Send(ctx, &types.AudioStreamMemberAudioEvent{Value: types.AudioEvent{AudioChunk: audio[:n]}}); err != nil {
			return err
		}
		audio = audio[n:]
	}
	return stream.Writer.Close()
}
