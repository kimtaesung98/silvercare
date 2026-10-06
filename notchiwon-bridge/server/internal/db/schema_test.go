package db_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/migrate"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

// newTestDB returns a migrated throwaway database, after checking that every
// migration's Down section undoes its Up (up → down to 0 → up).
func newTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pool := testdb.Pool(t, testdb.URL(t))
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	p, err := migrate.NewProvider(sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up: %v", err)
	}
	if _, err := p.DownTo(ctx, 0); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	return pool
}

type fixture struct {
	elderID uuid.UUID
	visitID uuid.UUID
}

func seed(t *testing.T, pool *pgxpool.Pool) fixture {
	t.Helper()
	f := testdb.Seed(t, pool, time.Now().Add(30*time.Minute))
	return fixture{elderID: f.ElderID, visitID: f.VisitID}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Two location updates arriving at the same time must start only one session.
func TestClaimVisitSessionOnlyOnce(t *testing.T) {
	pool := newTestDB(t)
	f := seed(t, pool)
	ctx := context.Background()

	const callers = 20
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		claimed int
	)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
				q := db.New(tx)
				visit, err := q.ClaimVisitSession(ctx, f.visitID)
				if err != nil {
					return err
				}
				_, err = q.CreatePickupSession(ctx, db.CreatePickupSessionParams{
					VisitID:           &visit.ID,
					ElderID:           visit.ElderID,
					TriggerEtaMinutes: ptr(int32(12)),
				})
				return err
			})
			switch {
			case err == nil:
				mu.Lock()
				claimed++
				mu.Unlock()
			case errors.Is(err, pgx.ErrNoRows):
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if claimed != 1 {
		t.Fatalf("claimed = %d, want 1", claimed)
	}
	var sessions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM conversation_session WHERE visit_id = $1`, f.visitID).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if sessions != 1 {
		t.Fatalf("sessions = %d, want 1", sessions)
	}
}

// An elder never has two open sessions, e.g. a companion button press during a pickup session.
func TestOneOpenSessionPerElder(t *testing.T) {
	pool := newTestDB(t)
	f := seed(t, pool)
	ctx := context.Background()
	q := db.New(pool)

	first, err := q.CreateCompanionSession(ctx, db.CreateCompanionSessionParams{ElderID: f.elderID, StartedBy: "ELDER"})
	if err != nil {
		t.Fatalf("first companion session: %v", err)
	}
	_, err = q.CreateCompanionSession(ctx, db.CreateCompanionSessionParams{ElderID: f.elderID, StartedBy: "SCHEDULE"})
	if !isUniqueViolation(err) {
		t.Fatalf("second open session: err = %v, want unique violation", err)
	}

	if _, err := q.EndSession(ctx, db.EndSessionParams{ID: first.ID, EndedReason: ptr("ELDER_DECLINED")}); err != nil {
		t.Fatalf("end session: %v", err)
	}
	if _, err := q.EndSession(ctx, db.EndSessionParams{ID: first.ID, EndedReason: ptr("TIMEOUT")}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("ending twice: err = %v, want no rows", err)
	}
	if _, err := q.CreateCompanionSession(ctx, db.CreateCompanionSessionParams{ElderID: f.elderID, StartedBy: "ELDER"}); err != nil {
		t.Fatalf("session after the first ended: %v", err)
	}
}

func TestSessionModeConstraints(t *testing.T) {
	pool := newTestDB(t)
	f := seed(t, pool)
	ctx := context.Background()

	cases := []struct {
		name string
		sql  string
		args []any
	}{
		{"pickup without visit", `INSERT INTO conversation_session (elder_id, mode, started_by, trigger_eta_minutes) VALUES ($1, 'PICKUP_BRIDGE', 'SYSTEM', 10)`, []any{f.elderID}},
		{"pickup without eta", `INSERT INTO conversation_session (visit_id, elder_id, mode, started_by) VALUES ($1, $2, 'PICKUP_BRIDGE', 'SYSTEM')`, []any{f.visitID, f.elderID}},
		{"companion with visit", `INSERT INTO conversation_session (visit_id, elder_id, mode, started_by) VALUES ($1, $2, 'COMPANION', 'ELDER')`, []any{f.visitID, f.elderID}},
		{"unknown mode", `INSERT INTO conversation_session (elder_id, mode, started_by) VALUES ($1, 'CHAT', 'ELDER')`, []any{f.elderID}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, tc.sql, tc.args...)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
				t.Fatalf("err = %v, want check violation", err)
			}
		})
	}
}

func TestBumpKeywordDecays(t *testing.T) {
	pool := newTestDB(t)
	f := seed(t, pool)
	ctx := context.Background()
	q := db.New(pool)

	now := time.Now()
	for _, mentions := range []int32{3, 2} {
		if _, err := q.BumpKeyword(ctx, db.BumpKeywordParams{
			ElderID: f.elderID, Keyword: "손주", Category: "FAMILY",
			Mentions: mentions, EmotionTone: "POSITIVE", MentionedAt: now,
		}); err != nil {
			t.Fatalf("bump: %v", err)
		}
	}
	top, err := q.ListTopKeywords(ctx, db.ListTopKeywordsParams{ElderID: f.elderID, MaxCount: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 1 {
		t.Fatalf("keywords = %d, want 1", len(top))
	}
	if got, want := top[0].Score, 3*0.9+2; fmt.Sprintf("%.4f", got) != fmt.Sprintf("%.4f", want) {
		t.Errorf("score = %v, want %v", got, want)
	}
	if top[0].MentionCountTotal != 5 {
		t.Errorf("mention_count_total = %d, want 5", top[0].MentionCountTotal)
	}
}

func ptr[T any](v T) *T { return &v }
