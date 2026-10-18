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

	// CompanionDailyTokenLimit caps one elder's companion talk per day when
	// the elder's own schedule sets no limit.
	CompanionDailyTokenLimit int64 `env:"COMPANION_DAILY_TOKEN_LIMIT" envDefault:"60000"`
	// CompanionIdleAfter ends a companion session the elder stopped answering.
	CompanionIdleAfter time.Duration `env:"COMPANION_IDLE_AFTER" envDefault:"5m"`
	// CompanionDigestAt is the local time the guardian's daily digest goes
	// out when the elder has no bedtime set.
	CompanionDigestAt string `env:"COMPANION_DIGEST_AT" envDefault:"21:00"`

	// SessionTriggerEtaMinutes starts a pickup session once the caregiver's ETA is at or below it.
	SessionTriggerEtaMinutes int `env:"SESSION_TRIGGER_ETA_MINUTES" envDefault:"15"`

	// TimeZone defines "today" for visit lists.
	TimeZone string `env:"TIME_ZONE" envDefault:"Asia/Seoul"`

	// AnthropicAPIKey is optional in development: without it every turn ends
	// with the fallback sentence.
	AnthropicAPIKey   string `env:"ANTHROPIC_API_KEY"`
	ConversationModel string `env:"CLAUDE_CONVERSATION_MODEL" envDefault:"claude-sonnet-5-5"`
	// SummaryModel writes arrival briefings and extracts keywords.
	SummaryModel string `env:"CLAUDE_SUMMARY_MODEL" envDefault:"claude-haiku-4-5"`

	// KakaoMobilityAPIKey is the Kakao REST API key for car directions (ETA).
	// Without it the ETA is the time left until the visit's scheduled time.
	KakaoMobilityAPIKey string `env:"KAKAO_MOBILITY_API_KEY"`

	// FCMCredentialsFile is the path of a Firebase service account key (JSON)
	// for escalation pushes. Without it escalations are only listed in the app.
	FCMCredentialsFile string `env:"FCM_CREDENTIALS_FILE"`

	// Conversation timings (internal/session.Config).
	FillerAfter          time.Duration `env:"CONVERSATION_FILLER_AFTER" envDefault:"2s"`
	FirstSentenceTimeout time.Duration `env:"CONVERSATION_FIRST_SENTENCE_TIMEOUT" envDefault:"6s"`
	TurnTimeout          time.Duration `env:"CONVERSATION_TURN_TIMEOUT" envDefault:"20s"`

	Speech Speech
}

// Speech is Amazon speech: Transcribe hears the elder (STT) and Polly voices
// AI sentences (TTS). Without AWS credentials the server has no speech: the
// tablet recognizes and speaks on device, and elder.audio fails with
// STT_FAILED. The AWS SDK reads the credentials itself (an `aws login`
// profile, or access keys); they are here to tell whether speech is on.
type Speech struct {
	Profile         string `env:"AWS_PROFILE"`
	AccessKeyID     string `env:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"AWS_SECRET_ACCESS_KEY"`
	Region          string `env:"AWS_REGION" envDefault:"ap-southeast-2"`
	// LanguageCode is the Transcribe language.
	LanguageCode string `env:"TRANSCRIBE_LANGUAGE" envDefault:"ko-KR"`
	// Voice is a Korean Polly voice: Seoyeon (standard, neural, generative)
	// or Jihye (neural).
	Voice  string `env:"POLLY_VOICE" envDefault:"Seoyeon"`
	Engine string `env:"POLLY_ENGINE" envDefault:"neural"`
	// Rate is an SSML prosody rate; elders follow a slightly slower voice
	// better. Empty or "100%" sends plain text.
	Rate string `env:"POLLY_RATE" envDefault:"90%"`
}

// Enabled reports whether AWS credentials are configured: a profile, or
// both access keys.
func (s Speech) Enabled() bool {
	return s.Profile != "" || (s.AccessKeyID != "" && s.SecretAccessKey != "")
}

// LoadSpeech reads only the speech settings (for cmd/admin).
func LoadSpeech() (Speech, error) { return env.ParseAs[Speech]() }

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
