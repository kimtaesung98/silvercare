// Package config loads server settings from environment variables.
package config

import (
	"errors"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds every setting the server reads at startup.
type Config struct {
	Port        int    `env:"PORT" envDefault:"8000"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	DatabaseURL string `env:"DATABASE_URL,required,notEmpty"`
	RedisURL    string `env:"REDIS_URL"`

	// MigrateOnStart applies pending goose migrations before serving.
	MigrateOnStart bool `env:"MIGRATE_ON_START" envDefault:"true"`

	// AuthSecret signs caregiver access tokens. At least 32 bytes.
	AuthSecret        string        `env:"AUTH_SECRET,required,notEmpty"`
	CaregiverTokenTTL time.Duration `env:"CAREGIVER_TOKEN_TTL" envDefault:"12h"`

	// SessionTriggerEtaMinutes starts a pickup session once the caregiver's ETA is at or below it.
	SessionTriggerEtaMinutes int `env:"SESSION_TRIGGER_ETA_MINUTES" envDefault:"15"`

	// TimeZone defines "today" for visit lists.
	TimeZone string `env:"TIME_ZONE" envDefault:"Asia/Seoul"`

	// AnthropicAPIKey is optional in development: without it every turn ends
	// with the fallback sentence.
	AnthropicAPIKey   string `env:"ANTHROPIC_API_KEY"`
	ConversationModel string `env:"CLAUDE_CONVERSATION_MODEL" envDefault:"claude-sonnet-5-5"`

	// Conversation timings (internal/session.Config).
	FillerAfter          time.Duration `env:"CONVERSATION_FILLER_AFTER" envDefault:"2s"`
	FirstSentenceTimeout time.Duration `env:"CONVERSATION_FIRST_SENTENCE_TIMEOUT" envDefault:"6s"`
	TurnTimeout          time.Duration `env:"CONVERSATION_TURN_TIMEOUT" envDefault:"20s"`
}

// Load reads Config from the process environment and checks it.
func Load() (Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return Config{}, err
	}
	if len(cfg.AuthSecret) < 32 {
		return Config{}, errors.New("AUTH_SECRET must be at least 32 bytes")
	}
	if cfg.SessionTriggerEtaMinutes < 0 {
		return Config{}, errors.New("SESSION_TRIGGER_ETA_MINUTES must not be negative")
	}
	if cfg.FillerAfter <= 0 || cfg.FirstSentenceTimeout <= 0 || cfg.TurnTimeout < cfg.FirstSentenceTimeout {
		return Config{}, errors.New("conversation timings must be positive, with CONVERSATION_TURN_TIMEOUT >= CONVERSATION_FIRST_SENTENCE_TIMEOUT")
	}
	if _, err := time.LoadLocation(cfg.TimeZone); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
