package speech

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/config"
)

// PollyAPI is the part of the Polly client Polly uses (a fake in tests).
type PollyAPI interface {
	SynthesizeSpeech(ctx context.Context, in *polly.SynthesizeSpeechInput, opts ...func(*polly.Options)) (*polly.SynthesizeSpeechOutput, error)
}

// Polly voices sentences with Amazon Polly.
type Polly struct {
	cfg    config.Polly
	client PollyAPI
}

// NewPolly returns a Polly that calls api.
func NewPolly(cfg config.Polly, api PollyAPI) *Polly {
	return &Polly{cfg: cfg, client: api}
}

// NewPollyFromEnv builds the AWS client from the standard credential chain
// (AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY, or a profile).
func NewPollyFromEnv(ctx context.Context, cfg config.Polly) (*Polly, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	return NewPolly(cfg, polly.NewFromConfig(awsCfg)), nil
}

// Synthesize implements Synthesizer. Polly has its own Korean voices, so the
// voice argument (a Clova speaker name or "default") is ignored and
// POLLY_VOICE is used.
func (p *Polly) Synthesize(ctx context.Context, text, _ string) ([]byte, error) {
	in := &polly.SynthesizeSpeechInput{
		Engine:       types.Engine(p.cfg.Engine),
		VoiceId:      types.VoiceId(p.cfg.VoiceID),
		LanguageCode: types.LanguageCodeKoKr,
		OutputFormat: types.OutputFormatMp3,
		Text:         aws.String(text),
		TextType:     types.TextTypeText,
	}
	if p.cfg.Rate != "" && p.cfg.Rate != "100%" {
		in.Text = aws.String(`<speak><prosody rate="` + html.EscapeString(p.cfg.Rate) + `">` + html.EscapeString(text) + `</prosody></speak>`)
		in.TextType = types.TextTypeSsml
	}
	out, err := p.client.SynthesizeSpeech(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("polly: %w", err)
	}
	defer out.AudioStream.Close()
	audio, err := io.ReadAll(io.LimitReader(out.AudioStream, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("polly audio: %w", err)
	}
	if len(audio) == 0 {
		return nil, errors.New("polly returned no audio")
	}
	return audio, nil
}

// NewSynthesizer returns the synthesizer TTS_PROVIDER names: Clova Voice
// (Unavailable without the Clova keys) or Amazon Polly.
func NewSynthesizer(ctx context.Context, s config.Speech) (Synthesizer, error) {
	if s.Voice.Provider == config.TTSPolly {
		return NewPollyFromEnv(ctx, s.Voice.Polly)
	}
	if s.Clova.Enabled() {
		return NewClova(ClovaConfig(s.Clova)), nil
	}
	return Unavailable{}, nil
}
