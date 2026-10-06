package companion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/riverqueue/river"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/jobs"
)

// Ended reasons the tick uses.
const (
	EndNoResponse = "NO_RESPONSE"
	EndBedtime    = "BEDTIME"
)

// Tablets is the tablet connections the tick needs: whether an elder's
// tablet is online, how to tell it about a session the server started, and
// how to end a session.
type Tablets interface {
	Connected(elderID uuid.UUID) bool
	CompanionStarted(ctx context.Context, s db.ConversationSession)
	End(ctx context.Context, sessionID uuid.UUID, reason string)
}

// Queue is where the tick puts the daily digests.
type Queue interface {
	Digest(ctx context.Context, elderID uuid.UUID, date string) error
}

// TickWorker runs every minute (jobs.CompanionTickArgs): it starts the
// check-in calls that have come due, ends sessions that went quiet or ran
// into bedtime, and queues each guardian's daily digest once their elder's
// day is over.
type TickWorker struct {
	river.WorkerDefaults[jobs.CompanionTickArgs]
	Queries *db.Queries
	Policy  Policy
	Tablets Tablets
	Queue   Queue
	Logger  *slog.Logger
	// IdleAfter ends a session the elder stopped answering.
	IdleAfter time.Duration
	// DigestAt is the local time of day the digest goes out when the elder
	// has no bedtime set.
	DigestAt Clock
	// PromptVersion is recorded on the sessions it starts.
	PromptVersion string
	Now           func() time.Time
}

// defaultIdleAfter is how long a companion session may stay quiet.
const defaultIdleAfter = 5 * time.Minute

func (w *TickWorker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

func (w *TickWorker) idleAfter() time.Duration {
	if w.IdleAfter > 0 {
		return w.IdleAfter
	}
	return defaultIdleAfter
}

// Timeout bounds one tick; it must finish well within the minute.
func (w *TickWorker) Timeout(*river.Job[jobs.CompanionTickArgs]) time.Duration {
	return 30 * time.Second
}

// Work implements river.Worker. A failing part is logged and the rest of the
// tick goes on: the next minute tries again.
func (w *TickWorker) Work(ctx context.Context, _ *river.Job[jobs.CompanionTickArgs]) error {
	now := w.now()
	if err := w.closeStale(ctx, now); err != nil {
		w.Logger.ErrorContext(ctx, "companion tick: close stale sessions failed", "err", err)
	}
	if err := w.startDue(ctx, now); err != nil {
		w.Logger.ErrorContext(ctx, "companion tick: start check-ins failed", "err", err)
	}
	if err := w.queueDigests(ctx, now); err != nil {
		w.Logger.ErrorContext(ctx, "companion tick: queue digests failed", "err", err)
	}
	return nil
}

// closeStale ends open companion sessions that went quiet or ran into bedtime.
func (w *TickWorker) closeStale(ctx context.Context, now time.Time) error {
	open, err := w.Queries.ListOpenCompanionSessions(ctx)
	if err != nil {
		return fmt.Errorf("list open companion sessions: %w", err)
	}
	for _, r := range open {
		s, err := w.Policy.Schedule(ctx, r.ConversationSession.ElderID)
		if err != nil {
			w.Logger.ErrorContext(ctx, "read schedule failed", "elder_id", r.ConversationSession.ElderID, "err", err)
			continue
		}
		switch {
		case s.InBedtime(now):
			w.Tablets.End(ctx, r.ConversationSession.ID, EndBedtime)
		case now.Sub(r.LastActivityAt) >= w.idleAfter():
			w.Tablets.End(ctx, r.ConversationSession.ID, EndNoResponse)
		}
	}
	return nil
}

// startDue starts a check-in call for every elder whose check-in time has
// just passed. The tablet must be connected: a call nobody hears would still
// spend the day's tokens.
func (w *TickWorker) startDue(ctx context.Context, now time.Time) error {
	rows, err := w.Queries.ListEnabledCompanionSchedules(ctx)
	if err != nil {
		return fmt.Errorf("list companion schedules: %w", err)
	}
	for _, row := range rows {
		s := FromRow(row, w.Policy.Location)
		due, ok := s.DueCheckIn(now, jobs.TickEvery*5)
		if !ok || !w.Tablets.Connected(s.ElderID) {
			continue
		}
		started, err := w.Queries.HasScheduledSessionSince(ctx, db.HasScheduledSessionSinceParams{
			ElderID: s.ElderID, Since: due,
		})
		if err != nil {
			w.Logger.ErrorContext(ctx, "read scheduled sessions failed", "elder_id", s.ElderID, "err", err)
			continue
		}
		if started {
			continue
		}
		why, err := w.Policy.canStart(ctx, s)
		if err != nil {
			w.Logger.ErrorContext(ctx, "check companion policy failed", "elder_id", s.ElderID, "err", err)
			continue
		}
		if why != "" {
			w.Logger.InfoContext(ctx, "check-in call skipped", "elder_id", s.ElderID, "reason", why, "due_at", due)
			continue
		}
		w.start(ctx, s.ElderID, due)
	}
	return nil
}

func (w *TickWorker) start(ctx context.Context, elderID uuid.UUID, due time.Time) {
	var version *string
	if w.PromptVersion != "" {
		version = &w.PromptVersion
	}
	sess, err := w.Queries.CreateCompanionSession(ctx, db.CreateCompanionSessionParams{
		ElderID: elderID, StartedBy: "SCHEDULE", PromptVersion: version,
	})
	var pgErr *pgconn.PgError
	switch {
	case err == nil:
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return // already talking (pickup bridge, or the elder pressed the button)
	default:
		w.Logger.ErrorContext(ctx, "start check-in call failed", "elder_id", elderID, "err", err)
		return
	}
	w.Logger.InfoContext(ctx, "check-in call started", "elder_id", elderID, "session_id", sess.ID, "due_at", due)
	w.Tablets.CompanionStarted(ctx, sess)
}

// queueDigests queues the guardian's daily digest once the elder's day of
// companion talk is over (bedtime, or DigestAt when there is no bedtime).
func (w *TickWorker) queueDigests(ctx context.Context, now time.Time) error {
	rows, err := w.Queries.ListEnabledCompanionSchedules(ctx)
	if err != nil {
		return fmt.Errorf("list companion schedules: %w", err)
	}
	for _, row := range rows {
		s := FromRow(row, w.Policy.Location)
		at := w.DigestAt
		if s.BedStart != nil {
			at = *s.BedStart
		}
		day := Schedule{ElderID: s.ElderID, CheckIns: []Clock{at}, Location: s.Location}
		if _, ok := day.DueCheckIn(now, jobs.TickEvery*5); !ok {
			continue
		}
		date := s.DayStart(now).Format(time.DateOnly)
		if err := w.Queue.Digest(ctx, s.ElderID, date); err != nil {
			w.Logger.ErrorContext(ctx, "queue daily digest failed", "elder_id", s.ElderID, "date", date, "err", err)
		}
	}
	return nil
}
