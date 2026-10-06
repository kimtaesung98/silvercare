// Package companion runs the after-daycare companion mode: when a companion
// session may start, the scheduled check-in calls, ending idle and bedtime
// sessions, the daily token limit, and the guardian's daily digest.
package companion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
)

// Reasons a companion session may not start (session.rejected reason).
const (
	Disabled   = "COMPANION_DISABLED"
	Bedtime    = "BEDTIME"
	TokenLimit = "TOKEN_LIMIT"
)

// Clock is a time of day in minutes after midnight.
type Clock int

// ClockOf converts a Postgres time.
func ClockOf(t pgtype.Time) (Clock, bool) {
	if !t.Valid {
		return 0, false
	}
	return Clock(t.Microseconds / int64(time.Minute/time.Microsecond)), true
}

// PgTime converts back.
func (c Clock) PgTime() pgtype.Time {
	return pgtype.Time{Microseconds: int64(c) * int64(time.Minute/time.Microsecond), Valid: true}
}

// ParseClock reads "HH:MM".
func ParseClock(s string) (Clock, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, fmt.Errorf("time of day %q: want HH:MM", s)
	}
	return Clock(t.Hour()*60 + t.Minute()), nil
}

func (c Clock) String() string { return fmt.Sprintf("%02d:%02d", int(c)/60, int(c)%60) }

func clockAt(t time.Time) Clock { return Clock(t.Hour()*60 + t.Minute()) }

// Schedule is an elder's companion settings with the time zone resolved.
type Schedule struct {
	ElderID    uuid.UUID
	Enabled    bool
	CheckIns   []Clock
	BedStart   *Clock
	BedEnd     *Clock
	Location   *time.Location
	TokenLimit *int64
}

// FromRow converts a stored schedule. An unknown time zone falls back to
// fallback.
func FromRow(r db.CompanionSchedule, fallback *time.Location) Schedule {
	s := Schedule{ElderID: r.ElderID, Enabled: r.Enabled, Location: fallback}
	if loc, err := time.LoadLocation(r.TimeZone); err == nil {
		s.Location = loc
	}
	for _, t := range r.CheckInTimes {
		if c, ok := ClockOf(t); ok {
			s.CheckIns = append(s.CheckIns, c)
		}
	}
	if a, ok := ClockOf(r.BedtimeStart); ok {
		if b, ok := ClockOf(r.BedtimeEnd); ok {
			s.BedStart, s.BedEnd = &a, &b
		}
	}
	if r.DailyTokenLimit != nil {
		n := int64(*r.DailyTokenLimit)
		s.TokenLimit = &n
	}
	return s
}

// Default is the schedule of an elder who has none: enabled, no check-ins,
// no bedtime.
func Default(elderID uuid.UUID, loc *time.Location) Schedule {
	return Schedule{ElderID: elderID, Enabled: true, Location: loc}
}

// InBedtime reports whether t falls in bedtime. Bedtime may cross midnight
// (21:00 to 07:00); start == end means no bedtime.
func (s Schedule) InBedtime(t time.Time) bool {
	if s.BedStart == nil || s.BedEnd == nil || *s.BedStart == *s.BedEnd {
		return false
	}
	c := clockAt(t.In(s.Location))
	a, b := *s.BedStart, *s.BedEnd
	if a < b {
		return c >= a && c < b
	}
	return c >= a || c < b
}

// DayStart is local midnight of t's day.
func (s Schedule) DayStart(t time.Time) time.Time {
	l := t.In(s.Location)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, s.Location)
}

// DueCheckIn returns the latest check-in instant in (now-window, now], if any.
// A tick that runs late still starts the call, but not one long past.
func (s Schedule) DueCheckIn(now time.Time, window time.Duration) (time.Time, bool) {
	var due time.Time
	day := s.DayStart(now)
	for _, d := range []time.Time{day.AddDate(0, 0, -1), day} {
		for _, c := range s.CheckIns {
			at := time.Date(d.Year(), d.Month(), d.Day(), int(c)/60, int(c)%60, 0, 0, s.Location)
			if !at.After(now) && now.Sub(at) < window && at.After(due) {
				due = at
			}
		}
	}
	return due, !due.IsZero()
}

// Policy decides whether a companion session may start or go on.
type Policy struct {
	Q *db.Queries
	// DefaultTokenLimit applies when the schedule sets none.
	DefaultTokenLimit int64
	Location          *time.Location
	Now               func() time.Time
}

func (p Policy) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

// Schedule reads an elder's schedule, or the default.
func (p Policy) Schedule(ctx context.Context, elderID uuid.UUID) (Schedule, error) {
	r, err := p.Q.GetCompanionSchedule(ctx, elderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Default(elderID, p.Location), nil
	}
	if err != nil {
		return Schedule{}, fmt.Errorf("read companion schedule: %w", err)
	}
	return FromRow(r, p.Location), nil
}

// Limit is the elder's daily token limit.
func (p Policy) Limit(s Schedule) int64 {
	if s.TokenLimit != nil {
		return *s.TokenLimit
	}
	return p.DefaultTokenLimit
}

// TokensToday is what the elder's conversations used today (local day).
func (p Policy) TokensToday(ctx context.Context, s Schedule) (int64, error) {
	n, err := p.Q.SumElderTokensSince(ctx, db.SumElderTokensSinceParams{
		ElderID: s.ElderID, Since: s.DayStart(p.now()),
	})
	if err != nil {
		return 0, fmt.Errorf("sum tokens: %w", err)
	}
	return n, nil
}

// CanStart returns "" when a companion session may start now, else the
// session.rejected reason.
func (p Policy) CanStart(ctx context.Context, elderID uuid.UUID) (string, error) {
	s, err := p.Schedule(ctx, elderID)
	if err != nil {
		return "", err
	}
	return p.canStart(ctx, s)
}

func (p Policy) canStart(ctx context.Context, s Schedule) (string, error) {
	switch {
	case !s.Enabled:
		return Disabled, nil
	case s.InBedtime(p.now()):
		return Bedtime, nil
	}
	used, err := p.TokensToday(ctx, s)
	if err != nil {
		return "", err
	}
	if used >= p.Limit(s) {
		return TokenLimit, nil
	}
	return "", nil
}

// Exhausted reports whether the elder used up today's tokens; the engine
// then closes the conversation instead of calling Claude.
func (p Policy) Exhausted(ctx context.Context, elderID uuid.UUID) (bool, error) {
	s, err := p.Schedule(ctx, elderID)
	if err != nil {
		return false, err
	}
	used, err := p.TokensToday(ctx, s)
	if err != nil {
		return false, err
	}
	return used >= p.Limit(s), nil
}
