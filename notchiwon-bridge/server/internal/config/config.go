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

	Clova Clova
}

// Clova is Naver Clova speech. Without the keys the server runs in text
// mode: elder.audio fails with STT_FAILED and replies carry no audio.
type Clova struct {
	ClientID       string `env:"NAVER_CLOVA_CLIENT_ID"`
	ClientSecret   string `env:"NAVER_CLOVA_CLIENT_SECRET"`
	STTURL         string `env:"CLOVA_STT_URL" envDefault:"https://naveropenapi.apigw.ntruss.com/recog/v1/stt"`
	TTSURL         string `env:"CLOVA_TTS_URL" envDefault:"https://naveropenapi.apigw.ntruss.com/tts-premium/v1/tts"`
	DefaultSpeaker string `env:"CLOVA_DEFAULT_SPEAKER" envDefault:"nara"`
	// TTSSpeed is -5 (faster) .. 5 (slower).
	TTSSpeed int `env:"CLOVA_TTS_SPEED" envDefault:"1"`
}

// Enabled reports whether both keys are set.
func (c Clova) Enabled() bool { return c.ClientID != "" && c.ClientSecret != "" }

// LoadClova reads only the Clova settings (for cmd/admin).
func LoadClova() (Clova, error) { return env.ParseAs[Clova]() }

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
	if cfg.Clova.TTSSpeed < -5 || cfg.Clova.TTSSpeed > 5 {
		return Config{}, errors.New("CLOVA_TTS_SPEED must be between -5 and 5")
	}
	if _, err := time.LoadLocation(cfg.TimeZone); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
