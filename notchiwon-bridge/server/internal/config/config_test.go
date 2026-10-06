package config

import (
	"strings"
	"testing"
	"time"
)

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("AUTH_SECRET", strings.Repeat("s", 32))
}

func TestLoadDefaults(t *testing.T) {
	setRequired(t)
	t.Setenv("PORT", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 8000 {
		t.Errorf("Port = %d, want 8000", cfg.Port)
	}
	if cfg.SessionTriggerEtaMinutes != 15 || !cfg.MigrateOnStart || cfg.CaregiverTokenTTL != 12*time.Hour || cfg.TimeZone != "Asia/Seoul" {
		t.Errorf("defaults = %+v", cfg)
	}
}

func TestLoadFromEnv(t *testing.T) {
	setRequired(t)
	t.Setenv("PORT", "9090")
	t.Setenv("SESSION_TRIGGER_ETA_MINUTES", "10")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 9090 || cfg.DatabaseURL != "postgres://example" || cfg.SessionTriggerEtaMinutes != 10 {
		t.Errorf("got %+v", cfg)
	}
}

func TestLoadRejects(t *testing.T) {
	cases := map[string]map[string]string{
		"missing database url": {"DATABASE_URL": ""},
		"short secret":         {"AUTH_SECRET": "short"},
		"negative trigger":     {"SESSION_TRIGGER_ETA_MINUTES": "-1"},
		"unknown time zone":    {"TIME_ZONE": "Mars/Base"},
	}
	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			setRequired(t)
			for k, v := range env {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Error("Load succeeded, want error")
			}
		})
	}
}
