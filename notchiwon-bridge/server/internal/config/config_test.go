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

func TestLoadSpeech(t *testing.T) {
	setRequired(t)
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_REGION", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	s := cfg.Speech
	if s.Enabled() || s.Region != "ap-southeast-2" || s.LanguageCode != "ko-KR" || s.Voice != "Seoyeon" || s.Engine != "neural" || s.Rate != "90%" {
		t.Errorf("speech defaults = %+v", s)
	}

	t.Setenv("AWS_PROFILE", "silvercare")
	if s, _ := LoadSpeech(); !s.Enabled() {
		t.Error("a profile alone should enable speech")
	}
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	if s, _ := LoadSpeech(); s.Enabled() {
		t.Error("one access key alone should not enable speech")
	}
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("POLLY_VOICE", "Jihye")
	s, err = LoadSpeech()
	if err != nil || !s.Enabled() || s.Voice != "Jihye" {
		t.Errorf("LoadSpeech = %+v, %v", s, err)
	}
}
