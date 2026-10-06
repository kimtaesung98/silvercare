package companion

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

var seoul = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic(err)
	}
	return loc
}()

func at(hour, minute int) time.Time {
	return time.Date(2026, 10, 6, hour, minute, 0, 0, seoul)
}

func TestInBedtimeCrossesMidnight(t *testing.T) {
	night := Schedule{Location: seoul, BedStart: clock(t, "21:00"), BedEnd: clock(t, "07:00")}
	day := Schedule{Location: seoul, BedStart: clock(t, "13:00"), BedEnd: clock(t, "15:00")}
	none := Schedule{Location: seoul}
	cases := []struct {
		name string
		s    Schedule
		t    time.Time
		want bool
	}{
		{"before bedtime", night, at(20, 59), false},
		{"bedtime starts", night, at(21, 0), true},
		{"after midnight", night, at(3, 0), true},
		{"bedtime ends", night, at(7, 0), false},
		{"afternoon nap", day, at(14, 0), true},
		{"outside nap", day, at(15, 0), false},
		{"no bedtime", none, at(3, 0), false},
	}
	for _, c := range cases {
		if got := c.s.InBedtime(c.t); got != c.want {
			t.Errorf("%s: InBedtime = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDueCheckIn(t *testing.T) {
	s := Schedule{Location: seoul, CheckIns: []Clock{*clock(t, "15:30"), *clock(t, "00:10")}}
	window := 5 * time.Minute

	if _, ok := s.DueCheckIn(at(15, 29), window); ok {
		t.Error("not due yet")
	}
	due, ok := s.DueCheckIn(at(15, 31), window)
	if !ok || !due.Equal(at(15, 30)) {
		t.Errorf("due = %v, %v, want 15:30", due, ok)
	}
	if _, ok := s.DueCheckIn(at(15, 40), window); ok {
		t.Error("long past: should not start late")
	}
	// Just after midnight the check-in of the day that just started counts.
	if due, ok := s.DueCheckIn(at(0, 12), window); !ok || !due.Equal(at(0, 10)) {
		t.Errorf("after midnight: %v, %v", due, ok)
	}
}

func clock(t *testing.T, s string) *Clock {
	t.Helper()
	c, err := ParseClock(s)
	if err != nil {
		t.Fatal(err)
	}
	return &c
}

func TestParseAndFormatClock(t *testing.T) {
	c, err := ParseClock("09:05")
	if err != nil || c != Clock(9*60+5) || c.String() != "09:05" {
		t.Fatalf("ParseClock = %v, %v, %q", c, err, c.String())
	}
	if back, ok := ClockOf(c.PgTime()); !ok || back != c {
		t.Errorf("round trip = %v, %v", back, ok)
	}
	if _, err := ParseClock("25:00"); err == nil {
		t.Error("25:00 should not parse")
	}
}

func policy(q *db.Queries, now time.Time, limit int64) Policy {
	return Policy{Q: q, DefaultTokenLimit: limit, Location: seoul, Now: func() time.Time { return now }}
}

func TestCanStart(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	ctx := context.Background()
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))

	// No schedule row: companion mode is on.
	p := policy(q, at(15, 0), 1000)
	if why, err := p.CanStart(ctx, f.ElderID); why != "" || err != nil {
		t.Fatalf("default: %q, %v", why, err)
	}

	testdb.CompanionSchedule(t, pool, f.ElderID, "Asia/Seoul", "21:00", "07:00", "15:30")
	if why, _ := policy(q, at(22, 0), 1000).CanStart(ctx, f.ElderID); why != Bedtime {
		t.Errorf("bedtime: %q", why)
	}
	if _, err := pool.Exec(ctx, `UPDATE companion_schedule SET enabled = false WHERE elder_id = $1`, f.ElderID); err != nil {
		t.Fatal(err)
	}
	if why, _ := p.CanStart(ctx, f.ElderID); why != Disabled {
		t.Errorf("disabled: %q", why)
	}
}

func TestCanStartTokenLimit(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	ctx := context.Background()
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	_, utts := testdb.CompanionTalk(t, pool, f, "ELDER_DECLINED",
		testdb.Said{Elder: true, Text: "오늘 날씨가 좋네"},
		testdb.Said{Text: "그러네요, 산책 다녀오셨어요?"})
	in, out := int32(400), int32(200)
	if err := q.SetUtteranceUsage(ctx, db.SetUtteranceUsageParams{
		ID: utts[1].ID, InputTokens: &in, OutputTokens: &out,
	}); err != nil {
		t.Fatal(err)
	}

	now := time.Now().In(seoul)
	p := Policy{Q: q, DefaultTokenLimit: 1000, Location: seoul, Now: func() time.Time { return now }}
	if used, err := p.TokensToday(ctx, Default(f.ElderID, seoul)); used != 600 || err != nil {
		t.Fatalf("tokens today = %d, %v", used, err)
	}
	if why, _ := p.CanStart(ctx, f.ElderID); why != "" {
		t.Errorf("under the limit: %q", why)
	}
	p.DefaultTokenLimit = 600
	if why, _ := p.CanStart(ctx, f.ElderID); why != TokenLimit {
		t.Errorf("at the limit: %q", why)
	}
	if over, err := p.Exhausted(ctx, f.ElderID); !over || err != nil {
		t.Errorf("exhausted = %v, %v", over, err)
	}

	// The elder's own limit wins over the server default.
	limit := int32(10000)
	if _, err := q.UpsertCompanionSchedule(ctx, db.UpsertCompanionScheduleParams{
		ElderID: f.ElderID, Enabled: true, TimeZone: "Asia/Seoul",
		CheckInTimes: []pgtype.Time{}, DailyTokenLimit: &limit,
	}); err != nil {
		t.Fatal(err)
	}
	if why, _ := p.CanStart(ctx, f.ElderID); why != "" {
		t.Errorf("own limit: %q", why)
	}
}

// fakeTablets records what the tick asked of the tablets.
type fakeTablets struct {
	online  map[uuid.UUID]bool
	started []db.ConversationSession
	ended   []struct {
		ID     uuid.UUID
		Reason string
	}
}

func (f *fakeTablets) Connected(elderID uuid.UUID) bool { return f.online[elderID] }

func (f *fakeTablets) CompanionStarted(_ context.Context, s db.ConversationSession) {
	f.started = append(f.started, s)
}

func (f *fakeTablets) End(_ context.Context, id uuid.UUID, reason string) {
	f.ended = append(f.ended, struct {
		ID     uuid.UUID
		Reason string
	}{id, reason})
}

// fakeQueue records the digests the tick queued.
type fakeQueue struct{ digests []string }

func (q *fakeQueue) Digest(_ context.Context, elderID uuid.UUID, date string) error {
	q.digests = append(q.digests, elderID.String()+"@"+date)
	return nil
}
