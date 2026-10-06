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

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/config"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/eta"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/httpapi"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/migrate"
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
	// The real ETA client (Kakao Mobility / TMAP) replaces eta.Schedule in stage 5,
	// and the /ws/elder handler becomes the session notifier in stage 3.
	visits := visit.NewService(pool, eta.Schedule{}, nil, visit.Config{
		TriggerEtaMinutes: cfg.SessionTriggerEtaMinutes,
		Location:          loc,
	}, logger)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.Port),
		Handler: httpapi.NewRouter(httpapi.Deps{
			Pool:   pool,
			Visits: visits,
			Tokens: auth.NewTokens(cfg.AuthSecret, cfg.CaregiverTokenTTL),
			Logger: logger,
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
	return srv.Shutdown(shutdownCtx)
}
