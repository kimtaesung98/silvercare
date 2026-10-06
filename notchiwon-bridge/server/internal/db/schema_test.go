package db_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/db/migrations"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
)

// newTestDB creates a throwaway database on the server named by
// TEST_DATABASE_URL, applies the migrations and drops it when the test ends.
// The test is skipped when TEST_DATABASE_URL is unset.
func newTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect admin: %v", err)
	}
	name := "nb_test_" + uuid.NewString()[:8]
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop database: %v", err)
		}
		_ = admin.Close(context.Background())
	})

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	u.Path = "/" + name
	dsn := u.String()

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	// up → down → up: the Down section must undo everything Up creates.
	for _, step := range []func(*sql.DB, string, ...goose.OptionsFunc) error{goose.Up, goose.Reset, goose.Up} {
		if err := step(sqlDB, "."); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type fixture struct {
	elderID uuid.UUID
	visitID uuid.UUID
}

func seed(t *testing.T, pool *pgxpool.Pool) fixture {
	t.Helper()
	ctx := context.Background()
	var f fixture
	err := pool.QueryRow(ctx, `
		WITH g AS (INSERT INTO guardian (name, phone) VALUES ('보호자', '010-0000-0000') RETURNING id),
		     c AS (INSERT INTO daycare_center (name) VALUES ('햇살센터') RETURNING id),
		     e AS (INSERT INTO elder (name, birth_date, dementia_stage, guardian_id, center_id)
		           SELECT '김순자', '1940-03-01', 'MILD', g.id, c.id FROM g, c RETURNING id, center_id),
		     cg AS (INSERT INTO caregiver (name, center_id) SELECT '이조무', e.center_id FROM e RETURNING id),
		     v AS (INSERT INTO visit (elder_id, caregiver_id, scheduled_time)
		           SELECT e.id, cg.id, now() + interval '30 minutes' FROM e, cg RETURNING id, elder_id)
		SELECT elder_id, id FROM v`).Scan(&f.elderID, &f.visitID)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return f
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
