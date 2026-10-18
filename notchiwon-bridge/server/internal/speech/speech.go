// Package speech turns elder audio into text (STT, Amazon Transcribe) and AI
// sentences into audio (TTS, Amazon Polly). The tablet only records and
// plays: it never holds an AWS key.
package speech

import (
	"context"
	"errors"
)

// Recognizer is speech to text.
type Recognizer interface {
	// Recognize returns what was said in audio. format is an elder.audio
	// format (pcm16, opus, aac); sampleRate applies to pcm16.
	Recognize(ctx context.Context, audio []byte, format string, sampleRate int) (string, error)
}

// Synthesizer is text to speech.
type Synthesizer interface {
	// Synthesize returns MP3 audio of text spoken in voice ("default" is the
	// server's default voice).
	Synthesize(ctx context.Context, text, voice string) ([]byte, error)
}

// ErrUnavailable means no AWS key is configured.
var ErrUnavailable = errors.New("speech is not configured (no AWS_PROFILE or AWS access keys)")

// Unavailable is used without AWS credentials: recognition fails and no audio is
// made, so the tablet recognizes and speaks on device.
type Unavailable struct{}

// Recognize implements Recognizer.
func (Unavailable) Recognize(context.Context, []byte, string, int) (string, error) {
	return "", ErrUnavailable
}

// Synthesize implements Synthesizer.
func (Unavailable) Synthesize(context.Context, string, string) ([]byte, error) {
	return nil, ErrUnavailable
}
