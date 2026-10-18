package speech

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/config"
)

// New returns Transcribe and Polly when AWS credentials are configured, and
// Unavailable for both otherwise. The SDK reads the profile or keys from the
// environment itself (the standard AWS credential chain).
func New(ctx context.Context, cfg config.Speech) (Recognizer, Synthesizer, error) {
	if !cfg.Enabled() {
		return Unavailable{}, Unavailable{}, nil
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Region))
	if err != nil {
		return nil, nil, fmt.Errorf("aws config: %w", err)
	}
	stt := NewTranscribe(cfg.LanguageCode, StreamStarter(transcribestreaming.NewFromConfig(awsCfg)))
	tts := NewPolly(cfg.Voice, cfg.Engine, cfg.Rate, polly.NewFromConfig(awsCfg))
	return stt, tts, nil
}
