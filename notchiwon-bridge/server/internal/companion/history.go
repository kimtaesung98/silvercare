package companion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/jobs"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/llm"
)

// HistoryWorker folds the early part of a long conversation into a summary
// (jobs.HistorySummaryArgs), so the next turns send that summary instead of
// the whole conversation.
type HistoryWorker struct {
	river.WorkerDefaults[jobs.HistorySummaryArgs]
	Queries   *db.Queries
	Compacter llm.Compacter
	Logger    *slog.Logger
}

// Timeout bounds one summary (a Haiku call and one write).
func (w *HistoryWorker) Timeout(*river.Job[jobs.HistorySummaryArgs]) time.Duration {
	return 30 * time.Second
}

// Work implements river.Worker.
func (w *HistoryWorker) Work(ctx context.Context, job *river.Job[jobs.HistorySummaryArgs]) error {
	id, through := job.Args.SessionID, job.Args.Through
	owner, err := w.Queries.GetSessionOwner(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(fmt.Errorf("session %s not found", id))
	}
	if err != nil {
		return fmt.Errorf("read session: %w", err)
	}
	sess := owner.ConversationSession
	if sess.HistorySummaryThrough != nil && *sess.HistorySummaryThrough >= through {
		return nil // a later job already summarized this much
	}

	utts, err := w.Queries.ListSessionUtterances(ctx, id)
	if err != nil {
		return fmt.Errorf("read utterances: %w", err)
	}
	in := llm.HistoryInput{ElderName: owner.ElderName}
	if sess.HistorySummary != nil {
		in.Previous = *sess.HistorySummary
	}
	from := int32(-1)
	if sess.HistorySummaryThrough != nil {
		from = *sess.HistorySummaryThrough
	}
	for _, u := range utts {
		if u.Seq <= from || u.Seq > through {
			continue
		}
		elder := u.Speaker == "ELDER"
		if n := len(in.Lines); n > 0 && !elder && !in.Lines[n-1].Elder {
			in.Lines[n-1].Text += " " + u.Text
			continue
		}
		in.Lines = append(in.Lines, llm.Line{Elder: elder, Text: u.Text})
	}
	if len(in.Lines) == 0 {
		return nil
	}

	summary, err := w.Compacter.Summarize(ctx, in)
	switch {
	case err == nil:
	case errors.Is(err, llm.ErrUnavailable):
		// Without Claude there is no summary; the conversation goes on with
		// the utterances it has.
		return nil
	default:
		return err // retried
	}
	if err := w.Queries.SetHistorySummary(ctx, db.SetHistorySummaryParams{
		ID: id, Summary: &summary, Through: &through,
	}); err != nil {
		return fmt.Errorf("save history summary: %w", err)
	}
	w.Logger.InfoContext(ctx, "history summarized", "session_id", id, "through", through, "lines", len(in.Lines))
	return nil
}
