// Package jobs is the background work queue (River on Postgres): the arrival
// briefing after a pickup session ends, and the push alert for every
// escalation event. Jobs survive a server restart and retry on failure.
//
// The job argument types live here; their workers live with the code they
// run (internal/briefing, internal/notify) and are registered in cmd/api.
package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
)

// Queues. Alerts have their own queue so a slow briefing never delays one.
const (
	QueueAlerts    = "alerts"
	QueueBriefings = "briefings"
)

// EscalationAlertArgs pushes an escalation to the people who must act on it.
type EscalationAlertArgs struct {
	EscalationID uuid.UUID `json:"escalation_id" river:"unique"`
}

// Kind implements river.JobArgs.
func (EscalationAlertArgs) Kind() string { return "escalation_alert" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (EscalationAlertArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueAlerts,
		// Retries back off (1s, 16s, 81s, ...); 6 attempts cover about 25 minutes.
		MaxAttempts: 6,
		UniqueOpts:  river.UniqueOpts{ByArgs: true},
	}
}

// BriefingArgs writes the arrival briefing of an ended pickup session.
type BriefingArgs struct {
	SessionID uuid.UUID `json:"session_id" river:"unique"`
}

// Kind implements river.JobArgs.
func (BriefingArgs) Kind() string { return "briefing" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (BriefingArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueBriefings,
		MaxAttempts: 5,
		UniqueOpts:  river.UniqueOpts{ByArgs: true},
	}
}

// NewClient returns a River client that works the given workers. Call Start
// to begin working; an unstarted client can still insert.
func NewClient(pool *pgxpool.Pool, workers *river.Workers, logger *slog.Logger) (*river.Client[pgx.Tx], error) {
	c, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			QueueAlerts:    {MaxWorkers: 4},
			QueueBriefings: {MaxWorkers: 2},
		},
		Workers: workers,
		Logger:  logger,
	})
	if err != nil {
		return nil, fmt.Errorf("river client: %w", err)
	}
	return c, nil
}

// NewInsertOnly returns a client that only inserts jobs (tests, cmd/admin).
func NewInsertOnly(pool *pgxpool.Pool) (*river.Client[pgx.Tx], error) {
	return river.NewClient(riverpgxv5.New(pool), &river.Config{})
}

// Enqueuer turns domain events into jobs. It implements session.Alerter and
// visit.SessionNotifier.
type Enqueuer struct {
	Client *river.Client[pgx.Tx]
	Logger *slog.Logger
}

// EscalationRaised queues the push alert for ev.
func (e Enqueuer) EscalationRaised(ctx context.Context, ev db.EscalationEvent) {
	if _, err := e.Client.Insert(ctx, EscalationAlertArgs{EscalationID: ev.ID}, nil); err != nil {
		// The event is saved and listed in the app's open escalations; the
		// push is what failed, so say so loudly.
		e.Logger.ErrorContext(ctx, "queue escalation alert failed", "escalation_id", ev.ID, "err", err)
	}
}

// SessionStarted implements visit.SessionNotifier.
func (Enqueuer) SessionStarted(context.Context, db.ConversationSession, int) {}

// EtaUpdated implements visit.SessionNotifier.
func (Enqueuer) EtaUpdated(context.Context, db.ConversationSession, int) {}

// SessionEnded queues the arrival briefing of a pickup session. Companion
// sessions get the guardian's daily digest instead (stage 6).
func (e Enqueuer) SessionEnded(ctx context.Context, s db.ConversationSession) {
	if s.Mode != "PICKUP_BRIDGE" {
		return
	}
	if _, err := e.Client.Insert(ctx, BriefingArgs{SessionID: s.ID}, nil); err != nil {
		e.Logger.ErrorContext(ctx, "queue briefing failed", "session_id", s.ID, "err", err)
	}
}
