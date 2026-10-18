package speech

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/polly/types"
)

// PollyAPI is the part of the Polly client Polly uses (a fake in tests).
type PollyAPI interface {
	SynthesizeSpeech(ctx context.Context, in *polly.SynthesizeSpeechInput, opts ...func(*polly.Options)) (*polly.SynthesizeSpeechOutput, error)
}

// Polly voices sentences with Amazon Polly as MP3.
type Polly struct {
	voice  types.VoiceId
	engine types.Engine
	rate   string
	api    PollyAPI
}

// NewPolly returns a Polly speaking with voice (Seoyeon or Jihye) on engine
// (neural, standard, generative). rate is an SSML prosody rate such as
// "90%"; empty or "100%" sends plain text.
func NewPolly(voice, engine, rate string, api PollyAPI) *Polly {
	return &Polly{voice: types.VoiceId(voice), engine: types.Engine(engine), rate: rate, api: api}
}

// koreanVoices are the Polly voices that speak Korean. An elder's
// preferred_tts_voice outside this set (or "default") uses POLLY_VOICE.
var koreanVoices = map[string]bool{"Seoyeon": true, "Jihye": true}

// Synthesize implements Synthesizer.
func (p *Polly) Synthesize(ctx context.Context, text, voice string) ([]byte, error) {
	in := &polly.SynthesizeSpeechInput{
		Engine:       p.engine,
		VoiceId:      p.voice,
		LanguageCode: types.LanguageCodeKoKr,
		OutputFormat: types.OutputFormatMp3,
		Text:         aws.String(text),
		TextType:     types.TextTypeText,
	}
	if koreanVoices[voice] {
		in.VoiceId = types.VoiceId(voice)
	}
	if p.rate != "" && p.rate != "100%" {
		in.Text = aws.String(`<speak><prosody rate="` + html.EscapeString(p.rate) + `">` + html.EscapeString(text) + `</prosody></speak>`)
		in.TextType = types.TextTypeSsml
	}
	out, err := p.api.SynthesizeSpeech(ctx, in)
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
