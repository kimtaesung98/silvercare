// Package testdb gives tests a throwaway, migrated Postgres database.
//
// Tests that need it are skipped unless TEST_DATABASE_URL points at a
// Postgres server the test may create and drop databases on (CI runs one).
package testdb

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/migrate"
)

// URL creates an empty database on the TEST_DATABASE_URL server, drops it
// when the test ends and returns its connection URL.
func URL(t testing.TB) string {
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
	return u.String()
}

// New returns a pool on a fresh database with every migration applied.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	pool := Pool(t, URL(t))
	if _, err := migrate.Up(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

// Pool opens a pool on dsn and closes it when the test ends.
func Pool(t testing.TB, dsn string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Fixture is one center with one elder, one caregiver and one visit between them.
type Fixture struct {
	CenterID    uuid.UUID
	GuardianID  uuid.UUID
	ElderID     uuid.UUID
	CaregiverID uuid.UUID
	VisitID     uuid.UUID
}

// Seed inserts a Fixture whose visit is scheduled at visitAt.
func Seed(t testing.TB, pool *pgxpool.Pool, visitAt time.Time) Fixture {
	t.Helper()
	var f Fixture
	err := pool.QueryRow(context.Background(), `
		WITH g AS (INSERT INTO guardian (name, phone) VALUES ('보호자', '010-0000-0000') RETURNING id),
		     c AS (INSERT INTO daycare_center (name) VALUES ('햇살센터') RETURNING id),
		     e AS (INSERT INTO elder (name, birth_date, dementia_stage, guardian_id, center_id)
		           SELECT '김순자', '1940-03-01', 'MILD', g.id, c.id FROM g, c RETURNING id, center_id, guardian_id),
		     cg AS (INSERT INTO caregiver (name, center_id) SELECT '이조무', e.center_id FROM e RETURNING id),
		     v AS (INSERT INTO visit (elder_id, caregiver_id, scheduled_time)
		           SELECT e.id, cg.id, $1 FROM e, cg RETURNING id)
		SELECT e.center_id, e.guardian_id, e.id, cg.id, v.id FROM e, cg, v`, visitAt,
	).Scan(&f.CenterID, &f.GuardianID, &f.ElderID, &f.CaregiverID, &f.VisitID)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return f
}
