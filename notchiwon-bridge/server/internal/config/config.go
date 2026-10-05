// Package config loads server settings from environment variables.
package config

import (
	"github.com/caarlos0/env/v11"
)

// Config holds every setting the server reads at startup.
type Config struct {
	Port        int    `env:"PORT" envDefault:"8000"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	DatabaseURL string `env:"DATABASE_URL"`
	RedisURL    string `env:"REDIS_URL"`
}

// Load reads Config from the process environment.
func Load() (Config, error) {
	return env.ParseAs[Config]()
}
