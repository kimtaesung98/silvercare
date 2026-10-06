package eta

import (
	"context"
	"testing"
	"time"
)

func TestScheduleMinutes(t *testing.T) {
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	s := Schedule{Now: func() time.Time { return now }}
	cases := map[time.Duration]int{
		20 * time.Minute:             20,
		90 * time.Second:             2,
		0:                            0,
		-5 * time.Minute:             0,
		15*time.Minute + time.Second: 16,
	}
	for until, want := range cases {
		got, err := s.Minutes(context.Background(), Request{ScheduledTime: now.Add(until)})
		if err != nil || got != want {
			t.Errorf("%v before: got %d, %v; want %d", until, got, err, want)
		}
	}
}
