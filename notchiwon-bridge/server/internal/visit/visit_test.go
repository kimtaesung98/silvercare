package visit

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/eta"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

type recorder struct {
	mu      sync.Mutex
	started []db.ConversationSession
	ended   []db.ConversationSession
}

func (r *recorder) SessionStarted(_ context.Context, s db.ConversationSession, _ int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.started = append(r.started, s)
}

func (r *recorder) SessionEnded(_ context.Context, s db.ConversationSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ended = append(r.ended, s)
}

var seoul = mustLoad("Asia/Seoul")

func mustLoad(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

// newService returns a Service whose clock is fixed at now, using the fake
// schedule ETA (minutes until the visit's scheduled time) and a 15-minute trigger.
func newService(pool *pgxpool.Pool, now time.Time) (*Service, *recorder) {
	rec := &recorder{}
	clock := func() time.Time { return now }
	s := NewService(pool, eta.Schedule{Now: clock}, rec,
		Config{TriggerEtaMinutes: 15, Location: seoul},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.now = clock
	return s, rec
}

var here = Location{Latitude: 37.5665, Longitude: 126.978, RecordedAt: time.Now()}

func TestRecordLocationStartsSessionAtTrigger(t *testing.T) {
	pool := testdb.New(t)
	now := time.Now().Truncate(time.Second)
	f := testdb.Seed(t, pool, now.Add(40*time.Minute))
	ctx := context.Background()

	s, rec := newService(pool, now)
	res, err := s.RecordLocation(ctx, f.CaregiverID, f.VisitID, here)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusEnRoute || res.EtaMinutes != 40 || res.SessionStarted || res.SessionID != nil {
		t.Fatalf("far away: %+v", res)
	}

	// 26 minutes later the ETA is 14 minutes, under the 15-minute trigger.
	s, rec = newService(pool, now.Add(26*time.Minute))
	res, err = s.RecordLocation(ctx, f.CaregiverID, f.VisitID, here)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != StatusSessionActive || res.EtaMinutes != 14 || !res.SessionStarted || res.SessionID == nil {
		t.Fatalf("at trigger: %+v", res)
	}
	if len(rec.started) != 1 || rec.started[0].ID != *res.SessionID || rec.started[0].Mode != "PICKUP_BRIDGE" {
		t.Fatalf("notified %+v", rec.started)
	}
	first := *res.SessionID

	res, err = s.RecordLocation(ctx, f.CaregiverID, f.VisitID, here)
	if err != nil {
		t.Fatal(err)
	}
	if res.SessionStarted || res.SessionID == nil || *res.SessionID != first {
		t.Fatalf("next ping: %+v", res)
	}
	if len(rec.started) != 1 {
		t.Errorf("started %d times", len(rec.started))
	}

	var locations int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM caregiver_location WHERE visit_id = $1`, f.VisitID).Scan(&locations); err != nil {
		t.Fatal(err)
	}
	if locations != 3 {
		t.Errorf("locations = %d, want 3", locations)
	}
}

// The stage 2 completion criterion: simultaneous location pings start one session.
func TestRecordLocationConcurrentStartsOneSession(t *testing.T) {
	pool := testdb.New(t)
	now := time.Now().Truncate(time.Second)
	f := testdb.Seed(t, pool, now.Add(5*time.Minute))
	s, rec := newService(pool, now)

	const callers = 20
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		started int
		ids     = map[uuid.UUID]bool{}
	)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := s.RecordLocation(context.Background(), f.CaregiverID, f.VisitID, here)
			if err != nil {
				t.Errorf("RecordLocation: %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			if res.SessionStarted {
				started++
			}
			if res.SessionID != nil {
				ids[*res.SessionID] = true
			}
		}()
	}
	wg.Wait()

	if started != 1 || len(rec.started) != 1 {
		t.Fatalf("sessionStarted true %d times, notified %d, want 1", started, len(rec.started))
	}
	if len(ids) != 1 {
		t.Fatalf("callers saw %d different session ids, want 1", len(ids))
	}
}

func TestRecordLocationPreemptsCompanionSession(t *testing.T) {
	pool := testdb.New(t)
	now := time.Now().Truncate(time.Second)
	f := testdb.Seed(t, pool, now.Add(10*time.Minute))
	ctx := context.Background()

	companion, err := db.New(pool).CreateCompanionSession(ctx, db.CreateCompanionSessionParams{ElderID: f.ElderID, StartedBy: "ELDER"})
	if err != nil {
		t.Fatal(err)
	}

	s, rec := newService(pool, now)
	res, err := s.RecordLocation(ctx, f.CaregiverID, f.VisitID, here)
	if err != nil {
		t.Fatal(err)
	}
	if !res.SessionStarted {
		t.Fatalf("%+v", res)
	}
	if len(rec.ended) != 1 || rec.ended[0].ID != companion.ID || *rec.ended[0].EndedReason != "PREEMPTED" {
		t.Fatalf("ended %+v", rec.ended)
	}
}

func TestOtherCaregiverCannotSeeVisit(t *testing.T) {
	pool := testdb.New(t)
	now := time.Now()
	f := testdb.Seed(t, pool, now.Add(10*time.Minute))
	other := testdb.Seed(t, pool, now.Add(10*time.Minute))
	s, _ := newService(pool, now)
	ctx := context.Background()

	if _, err := s.Get(ctx, other.CaregiverID, f.VisitID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get: %v", err)
	}
	if _, err := s.RecordLocation(ctx, other.CaregiverID, f.VisitID, here); !errors.Is(err, ErrNotFound) {
		t.Errorf("RecordLocation: %v", err)
	}
	if _, err := s.Arrive(ctx, other.CaregiverID, f.VisitID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Arrive: %v", err)
	}
	if _, err := s.Get(ctx, f.CaregiverID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get unknown: %v", err)
	}
}

func TestArriveEndsSession(t *testing.T) {
	pool := testdb.New(t)
	now := time.Now().Truncate(time.Second)
	f := testdb.Seed(t, pool, now.Add(5*time.Minute))
	ctx := context.Background()
	s, rec := newService(pool, now)

	res, err := s.RecordLocation(ctx, f.CaregiverID, f.VisitID, here)
	if err != nil || !res.SessionStarted {
		t.Fatalf("%+v %v", res, err)
	}

	d, err := s.Arrive(ctx, f.CaregiverID, f.VisitID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Visit.Status != StatusCompleted || d.Visit.ActualArrivalTime == nil || d.SessionID == nil {
		t.Fatalf("%+v", d)
	}
	if len(rec.ended) != 1 || *rec.ended[0].EndedReason != "CAREGIVER_ARRIVED" {
		t.Fatalf("ended %+v", rec.ended)
	}

	if _, err := s.Arrive(ctx, f.CaregiverID, f.VisitID); !errors.Is(err, ErrClosed) {
		t.Errorf("arrive twice: %v", err)
	}
	if _, err := s.RecordLocation(ctx, f.CaregiverID, f.VisitID, here); !errors.Is(err, ErrClosed) {
		t.Errorf("location after arrival: %v", err)
	}
}

func TestArriveWithoutSession(t *testing.T) {
	pool := testdb.New(t)
	now := time.Now()
	f := testdb.Seed(t, pool, now.Add(time.Hour))
	s, rec := newService(pool, now)

	d, err := s.Arrive(context.Background(), f.CaregiverID, f.VisitID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Visit.Status != StatusCompleted || d.SessionID != nil || len(rec.ended) != 0 {
		t.Fatalf("%+v %+v", d, rec.ended)
	}
}

func TestListTodayUsesLocalDay(t *testing.T) {
	pool := testdb.New(t)
	// 2026-10-06 10:00 in Seoul.
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, seoul)
	f := testdb.Seed(t, pool, time.Date(2026, 10, 6, 23, 30, 0, 0, seoul))
	ctx := context.Background()

	insert := func(at time.Time) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO visit (elder_id, caregiver_id, scheduled_time) VALUES ($1, $2, $3)`,
			f.ElderID, f.CaregiverID, at); err != nil {
			t.Fatal(err)
		}
	}
	insert(time.Date(2026, 10, 6, 0, 10, 0, 0, seoul))  // today, early (still 10-05 in UTC)
	insert(time.Date(2026, 10, 7, 0, 30, 0, 0, seoul))  // tomorrow
	insert(time.Date(2026, 10, 5, 23, 50, 0, 0, seoul)) // yesterday

	s, _ := newService(pool, now)
	visits, err := s.ListToday(ctx, f.CaregiverID)
	if err != nil {
		t.Fatal(err)
	}
	if len(visits) != 2 {
		t.Fatalf("got %d visits, want 2", len(visits))
	}
	if visits[0].Visit.ScheduledTime.After(visits[1].Visit.ScheduledTime) || visits[1].Visit.ID != f.VisitID {
		t.Errorf("order: %v, %v", visits[0].Visit.ScheduledTime, visits[1].Visit.ScheduledTime)
	}
	if visits[1].ElderName != "김순자" {
		t.Errorf("elder name %q", visits[1].ElderName)
	}
}
