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
	if _, err := time.LoadLocation(cfg.TimeZone); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
