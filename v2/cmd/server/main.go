// server는 v2 채팅 바탕과 말동무 AI를 띄웁니다.
//
//	go run ./cmd/server            # http://localhost:8080
//
// ANTHROPIC_API_KEY가 있으면 Claude가, 없으면 정해진 맞장구가 답합니다.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/kimtaesung98/silvercare/v2/companion"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	addr := envOr("ADDR", ":8080")
	key := os.Getenv("ANTHROPIC_API_KEY")
	model := envOr("COMPANION_MODEL", "claude-sonnet-5-5")

	newBrain := func() companion.Brain { return &companion.SimpleBrain{} }
	if key != "" {
		newBrain = func() companion.Brain { return &companion.ClaudeBrain{APIKey: key, Model: model} }
		log.Info("말동무: Claude", "model", model)
	} else {
		log.Warn("ANTHROPIC_API_KEY가 없어 말동무가 정해진 맞장구로 답합니다")
	}

	s := newServer(ctx, newBrain, log)
	srv := &http.Server{Addr: addr, Handler: s.routes()}
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	log.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("server", "err", err)
		os.Exit(1)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
