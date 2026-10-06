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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
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

// Said is one utterance for Conversation: Elder true for the elder, false for the AI.
type Said struct {
	Elder bool
	Text  string
}

// Conversation starts the fixture's pickup session, records the lines (one
// turn each) and, when end is true, ends it as CAREGIVER_ARRIVED.
func Conversation(t testing.TB, pool *pgxpool.Pool, f Fixture, end bool, lines ...Said) (db.ConversationSession, []db.Utterance) {
	t.Helper()
	ctx := context.Background()
	q := db.New(pool)
	eta := int32(10)
	sess, err := q.CreatePickupSession(ctx, db.CreatePickupSessionParams{VisitID: &f.VisitID, ElderID: f.ElderID, TriggerEtaMinutes: &eta})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	utts := make([]db.Utterance, len(lines))
	for i, l := range lines {
		speaker, chunk := "AI", int32(1)
		if l.Elder {
			speaker, chunk = "ELDER", 0
		}
		utts[i], err = q.CreateUtterance(ctx, db.CreateUtteranceParams{SessionID: sess.ID, Seq: int32(i), ChunkIndex: chunk, Speaker: speaker, Text: l.Text})
		if err != nil {
			t.Fatalf("create utterance: %v", err)
		}
	}
	if end {
		reason := "CAREGIVER_ARRIVED"
		if sess, err = q.EndSession(ctx, db.EndSessionParams{ID: sess.ID, EndedReason: &reason}); err != nil {
			t.Fatalf("end session: %v", err)
		}
	}
	return sess, utts
}

// Phone registers a caregiver phone for push alerts.
func Phone(t testing.TB, pool *pgxpool.Pool, caregiverID uuid.UUID, fcmToken string) db.Device {
	t.Helper()
	d, err := db.New(pool).RegisterCaregiverPhone(context.Background(), db.RegisterCaregiverPhoneParams{CaregiverID: &caregiverID, FcmToken: &fcmToken})
	if err != nil {
		t.Fatalf("register phone: %v", err)
	}
	return d
}

// GuardianPhone registers a guardian phone for push alerts.
func GuardianPhone(t testing.TB, pool *pgxpool.Pool, guardianID uuid.UUID, fcmToken string) db.Device {
	t.Helper()
	d, err := db.New(pool).RegisterGuardianPhone(context.Background(), db.RegisterGuardianPhoneParams{
		GuardianID: &guardianID, FcmToken: &fcmToken,
	})
	if err != nil {
		t.Fatalf("register guardian phone: %v", err)
	}
	return d
}

// CompanionTalk starts a companion session for the fixture's elder, records
// the lines (one turn each) and ends it with reason when end is true.
func CompanionTalk(t testing.TB, pool *pgxpool.Pool, f Fixture, end string, lines ...Said) (db.ConversationSession, []db.Utterance) {
	t.Helper()
	ctx := context.Background()
	q := db.New(pool)
	sess, err := q.CreateCompanionSession(ctx, db.CreateCompanionSessionParams{ElderID: f.ElderID, StartedBy: "ELDER"})
	if err != nil {
		t.Fatalf("create companion session: %v", err)
	}
	utts := make([]db.Utterance, len(lines))
	for i, l := range lines {
		speaker, chunk := "AI", int32(1)
		if l.Elder {
			speaker, chunk = "ELDER", 0
		}
		utts[i], err = q.CreateUtterance(ctx, db.CreateUtteranceParams{
			SessionID: sess.ID, Seq: int32(i), ChunkIndex: chunk, Speaker: speaker, Text: l.Text,
		})
		if err != nil {
			t.Fatalf("create utterance: %v", err)
		}
	}
	if end != "" {
		if sess, err = q.EndSession(ctx, db.EndSessionParams{ID: sess.ID, EndedReason: &end}); err != nil {
			t.Fatalf("end session: %v", err)
		}
	}
	return sess, utts
}

// CompanionSchedule stores an elder's companion settings. checkIns and the
// bedtime are "HH:MM"; an empty bedtime means none.
func CompanionSchedule(t testing.TB, pool *pgxpool.Pool, elderID uuid.UUID, tz string, bedStart, bedEnd string, checkIns ...string) db.CompanionSchedule {
	t.Helper()
	clock := func(s string) pgtype.Time {
		at, err := time.Parse("15:04", s)
		if err != nil {
			t.Fatalf("time of day %q: %v", s, err)
		}
		return pgtype.Time{Microseconds: int64(at.Hour()*60+at.Minute()) * int64(time.Minute/time.Microsecond), Valid: true}
	}
	params := db.UpsertCompanionScheduleParams{
		ElderID: elderID, Enabled: true, TimeZone: tz, CheckInTimes: []pgtype.Time{},
	}
	for _, c := range checkIns {
		params.CheckInTimes = append(params.CheckInTimes, clock(c))
	}
	if bedStart != "" {
		params.BedtimeStart, params.BedtimeEnd = clock(bedStart), clock(bedEnd)
	}
	row, err := db.New(pool).UpsertCompanionSchedule(context.Background(), params)
	if err != nil {
		t.Fatalf("save companion schedule: %v", err)
	}
	return row
}

// GuardianLogin gives the fixture's guardian a login and returns it.
func GuardianLogin(t testing.TB, pool *pgxpool.Pool, guardianID uuid.UUID, loginID, password string) {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.New(pool).SetGuardianLogin(context.Background(), db.SetGuardianLoginParams{
		ID: guardianID, LoginID: &loginID, PasswordHash: &hash,
	}); err != nil {
		t.Fatalf("set guardian login: %v", err)
	}
}
