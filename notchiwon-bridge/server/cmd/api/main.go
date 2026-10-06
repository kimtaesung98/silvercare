// Command api runs the Notchiwon bridge HTTP and WebSocket server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/briefing"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/config"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/eta"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/httpapi"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/jobs"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/llm"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/migrate"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/notify"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/opener"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/session"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/speech"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/visit"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return fmt.Errorf("parse LOG_LEVEL: %w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	if cfg.MigrateOnStart {
		applied, err := migrate.Up(ctx, pool)
		if err != nil {
			return err
		}
		logger.Info("migrations applied", "versions", applied)
	}

	loc, err := time.LoadLocation(cfg.TimeZone)
	if err != nil {
		return err
	}
	var (
		claude     llm.Client     = llm.Unavailable{}
		summarizer llm.Summarizer = llm.Unavailable{}
	)
	if cfg.AnthropicAPIKey != "" {
		claude = llm.NewAnthropic(cfg.AnthropicAPIKey, cfg.ConversationModel)
		summarizer = llm.NewAnthropicSummarizer(llm.NewAnthropic(cfg.AnthropicAPIKey, cfg.SummaryModel))
	} else {
		logger.Warn("ANTHROPIC_API_KEY is empty: every turn ends with the fallback sentence, briefings quote the elder")
	}
	q := db.New(pool)
	openers := opener.NewLibrary(q)
	engine := session.NewEngine(q, claude, openers, session.Config{
		FillerAfter:          cfg.FillerAfter,
		FirstSentenceTimeout: cfg.FirstSentenceTimeout,
		TurnTimeout:          cfg.TurnTimeout,
		MaxSentences:         session.DefaultConfig.MaxSentences,
	}, logger)
	var (
		stt speech.Recognizer  = speech.Unavailable{}
		tts speech.Synthesizer = speech.Unavailable{}
	)
	if cfg.Clova.Enabled() {
		clova := speech.NewClova(speech.ClovaConfig(cfg.Clova))
		stt, tts = clova, clova
	} else {
		logger.Warn("NAVER_CLOVA_CLIENT_ID/SECRET are empty: text mode only (no speech recognition or synthesis)")
	}
	hub, err := session.NewHub(q, engine, openers, session.HubConfig{PromptVersion: llm.PromptVersion, STT: stt, TTS: tts}, logger)
	if err != nil {
		return err
	}
	logger.Info("conversation engine ready", "model", cfg.ConversationModel, "prompt_version", llm.PromptVersion)

	// Background jobs: escalation pushes and arrival briefings.
	var sender notify.Sender = notify.Unavailable{}
	if cfg.FCMCredentialsFile != "" {
		key, err := os.ReadFile(cfg.FCMCredentialsFile)
		if err != nil {
			return fmt.Errorf("read FCM_CREDENTIALS_FILE: %w", err)
		}
		fcm, err := notify.NewFCM(ctx, key)
		if err != nil {
			return err
		}
		sender = fcm
	} else {
		logger.Warn("FCM_CREDENTIALS_FILE is empty: escalations are not pushed, only listed in the caregiver app")
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, &notify.EscalationWorker{Queries: q, Sender: sender, Logger: logger})
	river.AddWorker(workers, &briefing.Worker{Pool: pool, Summarizer: summarizer, Logger: logger})
	queue, err := jobs.NewClient(pool, workers, logger)
	if err != nil {
		return err
	}
	if err := queue.Start(ctx); err != nil {
		return fmt.Errorf("start job queue: %w", err)
	}
	enqueuer := jobs.Enqueuer{Client: queue, Logger: logger}
	engine.SetAlerter(enqueuer)

	var estimator eta.Estimator = eta.Schedule{}
	if cfg.KakaoMobilityAPIKey != "" {
		estimator = eta.Kakao{APIKey: cfg.KakaoMobilityAPIKey, Logger: logger}
	} else {
		logger.Warn("KAKAO_MOBILITY_API_KEY is empty: ETA is the time left until the scheduled visit")
	}
	visits := visit.NewService(pool, estimator, visit.Notifiers{hub, enqueuer}, visit.Config{
		TriggerEtaMinutes: cfg.SessionTriggerEtaMinutes,
		Location:          loc,
		PromptVersion:     llm.PromptVersion,
	}, logger)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.Port),
		Handler: httpapi.NewRouter(httpapi.Deps{
			Pool:    pool,
			Visits:  visits,
			Tokens:  auth.NewTokens(cfg.AuthSecret, cfg.CaregiverTokenTTL),
			Openers: openers,
			TTS:     tts,
			ElderWS: hub,
			Logger:  logger,
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = srv.Shutdown(shutdownCtx)
	// Let running jobs finish; unfinished ones are picked up on the next start.
	if qerr := queue.Stop(shutdownCtx); qerr != nil {
		logger.Error("stop job queue", "err", qerr)
	}
	return err
}
