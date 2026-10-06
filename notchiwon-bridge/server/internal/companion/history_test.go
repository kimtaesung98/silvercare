package companion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/jobs"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/llm"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

// fakeCompacter records what it was asked to summarize.
type fakeCompacter struct {
	calls []llm.HistoryInput
	out   string
	err   error
}

func (f *fakeCompacter) Summarize(_ context.Context, in llm.HistoryInput) (string, error) {
	f.calls = append(f.calls, in)
	if f.err != nil {
		return "", f.err
	}
	return f.out, nil
}

func summarize(t *testing.T, w *HistoryWorker, id uuid.UUID, through int32) error {
	t.Helper()
	return w.Work(context.Background(), &river.Job[jobs.HistorySummaryArgs]{
		Args: jobs.HistorySummaryArgs{SessionID: id, Through: through},
	})
}

func TestHistorySummaryFoldsEarlyTurns(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	ctx := context.Background()
	f := testdb.Seed(t, pool, time.Now().Add(24*time.Hour))
	sess, _ := testdb.CompanionTalk(t, pool, f, "",
		testdb.Said{Elder: true, Text: "큰아들이 어제 왔어"},
		testdb.Said{Text: "반가우셨겠어요."},
		testdb.Said{Text: "무슨 이야기를 하셨어요?"},
		testdb.Said{Elder: true, Text: "손녀 이야기를 했지"},
		testdb.Said{Text: "손녀 이야기라니 좋네요."},
	)

	c := &fakeCompacter{out: "큰아들 방문과 손녀 이야기를 나누셨다."}
	w := &HistoryWorker{Queries: q, Compacter: c, Logger: discard}
	if err := summarize(t, w, sess.ID, 2); err != nil {
		t.Fatal(err)
	}
	if len(c.calls) != 1 {
		t.Fatalf("calls = %d", len(c.calls))
	}
	in := c.calls[0]
	if in.ElderName != "김순자" || in.Previous != "" {
		t.Errorf("input = %+v", in)
	}
	// The two AI chunks are one line, so three utterances make two lines.
	if len(in.Lines) != 2 || !in.Lines[0].Elder || in.Lines[1].Text != "반가우셨겠어요. 무슨 이야기를 하셨어요?" {
		t.Fatalf("lines = %+v", in.Lines)
	}

	row, err := q.GetSessionOwner(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s := row.ConversationSession; s.HistorySummary == nil || *s.HistorySummary != c.out ||
		s.HistorySummaryThrough == nil || *s.HistorySummaryThrough != 2 {
		t.Fatalf("stored = %+v", row.ConversationSession)
	}

	// The next summary carries the first one and only the new turns.
	c.out = "큰아들과 손녀 이야기를 나누고, 기분이 좋으셨다."
	if err := summarize(t, w, sess.ID, 4); err != nil {
		t.Fatal(err)
	}
	in = c.calls[1]
	if in.Previous != "큰아들 방문과 손녀 이야기를 나누셨다." || len(in.Lines) != 2 || in.Lines[0].Text != "손녀 이야기를 했지" {
		t.Fatalf("second input = %+v", in)
	}

	// A job that would go backwards neither asks Claude nor rewinds the row.
	if err := summarize(t, w, sess.ID, 2); err != nil {
		t.Fatal(err)
	}
	if len(c.calls) != 2 {
		t.Errorf("a stale job summarized again: %d calls", len(c.calls))
	}
	row, err = q.GetSessionOwner(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if *row.ConversationSession.HistorySummaryThrough != 4 {
		t.Errorf("through = %d, want 4", *row.ConversationSession.HistorySummaryThrough)
	}
}

func TestHistorySummaryWithoutClaude(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	f := testdb.Seed(t, pool, time.Now().Add(24*time.Hour))
	sess, _ := testdb.CompanionTalk(t, pool, f, "", testdb.Said{Elder: true, Text: "날씨가 좋네"})

	w := &HistoryWorker{Queries: q, Compacter: &fakeCompacter{err: llm.ErrUnavailable}, Logger: discard}
	if err := summarize(t, w, sess.ID, 0); err != nil {
		t.Fatalf("no key should not fail the job: %v", err)
	}
	row, err := q.GetSessionOwner(context.Background(), sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if row.ConversationSession.HistorySummary != nil {
		t.Errorf("summary written without Claude: %v", *row.ConversationSession.HistorySummary)
	}

	// A real failure is retried.
	boom := errors.New("overloaded")
	w.Compacter = &fakeCompacter{err: boom}
	if err := summarize(t, w, sess.ID, 0); !errors.Is(err, boom) {
		t.Errorf("err = %v, want it retried", err)
	}

	// A session that is gone is cancelled, not retried forever.
	var cancel *river.JobCancelError
	if err := summarize(t, w, uuid.New(), 0); !errors.As(err, &cancel) {
		t.Errorf("missing session err = %v, want it cancelled", err)
	}
}

func TestHistorySummaryNoNewTurns(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	f := testdb.Seed(t, pool, time.Now().Add(24*time.Hour))
	sess, _ := testdb.CompanionTalk(t, pool, f, "")

	c := &fakeCompacter{out: "..."}
	w := &HistoryWorker{Queries: q, Compacter: c, Logger: discard}
	if err := summarize(t, w, sess.ID, 5); err != nil {
		t.Fatal(err)
	}
	if len(c.calls) != 0 {
		t.Errorf("summarized an empty conversation: %+v", c.calls)
	}
}
