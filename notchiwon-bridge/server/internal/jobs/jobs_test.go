package jobs

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

func queued(t *testing.T, pool *pgxpool.Pool) map[string]int {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT kind, queue, count(*) FROM river_job GROUP BY kind, queue`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var kind, queue string
		var n int
		if err := rows.Scan(&kind, &queue, &n); err != nil {
			t.Fatal(err)
		}
		out[kind+"@"+queue] = n
	}
	return out
}

func TestEnqueuer(t *testing.T) {
	pool := testdb.New(t)
	client, err := NewInsertOnly(pool)
	if err != nil {
		t.Fatal(err)
	}
	e := Enqueuer{Client: client, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	ctx := context.Background()
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	sess, _ := testdb.Conversation(t, pool, f, true)

	ev := db.EscalationEvent{ID: uuid.New()}
	e.EscalationRaised(ctx, ev)
	e.EscalationRaised(ctx, ev) // the same event twice is one job
	e.SessionEnded(ctx, sess)
	e.SessionEnded(ctx, sess)
	e.SessionEnded(ctx, db.ConversationSession{ID: uuid.New(), Mode: "COMPANION"}) // no briefing
	e.SessionStarted(ctx, sess, 10)
	e.EtaUpdated(ctx, sess, 5)

	got := queued(t, pool)
	want := map[string]int{"escalation_alert@alerts": 1, "briefing@briefings": 1}
	if len(got) != len(want) || got["escalation_alert@alerts"] != 1 || got["briefing@briefings"] != 1 {
		t.Errorf("queued = %v, want %v", got, want)
	}
}
