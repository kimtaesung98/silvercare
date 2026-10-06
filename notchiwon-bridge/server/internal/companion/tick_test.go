package companion

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/jobs"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

func tick(t *testing.T, w *TickWorker) {
	t.Helper()
	if err := w.Work(context.Background(), &river.Job[jobs.CompanionTickArgs]{}); err != nil {
		t.Fatal(err)
	}
}

func TestTickStartsCheckInCall(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	ctx := context.Background()
	f := testdb.Seed(t, pool, time.Now().Add(24*time.Hour))
	testdb.CompanionSchedule(t, pool, f.ElderID, "Asia/Seoul", "21:00", "07:00", "15:30")

	tablets := &fakeTablets{online: map[uuid.UUID]bool{}}
	queue := &fakeQueue{}
	w := &TickWorker{
		Queries: q, Policy: policy(q, at(15, 31), 100000), Tablets: tablets, Queue: queue,
		Logger: discard, DigestAt: *clock(t, "21:00"), Now: func() time.Time { return at(15, 31) },
	}

	// The tablet is off: a call nobody hears would spend the day's tokens.
	tick(t, w)
	if len(tablets.started) != 0 {
		t.Fatalf("started %d calls with the tablet offline", len(tablets.started))
	}

	tablets.online[f.ElderID] = true
	tick(t, w)
	if len(tablets.started) != 1 || tablets.started[0].StartedBy != "SCHEDULE" {
		t.Fatalf("started = %+v", tablets.started)
	}
	sess := tablets.started[0]
	if sess.Mode != "COMPANION" || sess.PromptVersion != nil {
		t.Errorf("session = %+v", sess)
	}

	// The same check-in time does not start a second call.
	tick(t, w)
	if len(tablets.started) != 1 {
		t.Errorf("started %d calls for one check-in time", len(tablets.started))
	}

	// Bedtime ends the call the tick itself started.
	w.Policy = policy(q, at(21, 1), 100000)
	w.Now = func() time.Time { return at(21, 1) }
	tick(t, w)
	if len(tablets.ended) != 1 || tablets.ended[0].Reason != EndBedtime || tablets.ended[0].ID != sess.ID {
		t.Errorf("ended = %+v", tablets.ended)
	}
	// Bedtime is also when the guardian's digest goes out.
	if len(queue.digests) != 1 || queue.digests[0] != f.ElderID.String()+"@2026-10-06" {
		t.Errorf("digests = %v", queue.digests)
	}
	if _, err := q.GetOpenSessionForElder(ctx, f.ElderID); err == nil {
		// fakeTablets only records; the session is still open in the database.
		t.Log("session still open: the hub ends it in production")
	}
}

func TestTickEndsQuietSession(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	f := testdb.Seed(t, pool, time.Now().Add(24*time.Hour))
	sess, _ := testdb.CompanionTalk(t, pool, f, "", testdb.Said{Elder: true, Text: "오늘 날씨가 좋네"})

	now := time.Now()
	tablets := &fakeTablets{online: map[uuid.UUID]bool{f.ElderID: true}}
	w := &TickWorker{
		Queries: q, Policy: Policy{Q: q, DefaultTokenLimit: 100000, Location: seoul, Now: func() time.Time { return now }},
		Tablets: tablets, Queue: &fakeQueue{}, Logger: discard, IdleAfter: time.Minute,
		DigestAt: *clock(t, "21:00"), Now: func() time.Time { return now },
	}
	tick(t, w)
	if len(tablets.ended) != 0 {
		t.Fatalf("ended a session that just spoke: %+v", tablets.ended)
	}

	later := now.Add(2 * time.Minute)
	w.Now = func() time.Time { return later }
	w.Policy.Now = w.Now
	tick(t, w)
	if len(tablets.ended) != 1 || tablets.ended[0].ID != sess.ID || tablets.ended[0].Reason != EndNoResponse {
		t.Errorf("ended = %+v", tablets.ended)
	}
}

func TestTickSkipsCallOverTokenLimit(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	ctx := context.Background()
	f := testdb.Seed(t, pool, time.Now().Add(24*time.Hour))
	testdb.CompanionSchedule(t, pool, f.ElderID, "Asia/Seoul", "", "", "15:30")
	_, utts := testdb.CompanionTalk(t, pool, f, "ELDER_DECLINED",
		testdb.Said{Elder: true, Text: "오늘 날씨가 좋네"},
		testdb.Said{Text: "그러네요."})
	in, out := int32(900), int32(200)
	if err := q.SetUtteranceUsage(ctx, db.SetUtteranceUsageParams{
		ID: utts[1].ID, InputTokens: &in, OutputTokens: &out,
	}); err != nil {
		t.Fatal(err)
	}

	// The utterances were written now, so count the day as today.
	now := time.Now().In(seoul)
	due := time.Date(now.Year(), now.Month(), now.Day(), 15, 30, 30, 0, seoul)
	if _, err := pool.Exec(ctx, `UPDATE companion_schedule SET check_in_times = ARRAY['15:30'::time] WHERE elder_id = $1`, f.ElderID); err != nil {
		t.Fatal(err)
	}
	tablets := &fakeTablets{online: map[uuid.UUID]bool{f.ElderID: true}}
	w := &TickWorker{
		Queries: q, Policy: Policy{Q: q, DefaultTokenLimit: 1000, Location: seoul, Now: func() time.Time { return due }},
		Tablets: tablets, Queue: &fakeQueue{}, Logger: discard,
		DigestAt: *clock(t, "21:00"), Now: func() time.Time { return due },
	}
	tick(t, w)
	if len(tablets.started) != 0 {
		t.Errorf("started a call over the token limit: %+v", tablets.started)
	}
}
