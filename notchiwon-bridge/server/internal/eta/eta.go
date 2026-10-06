// Package eta estimates how many minutes a caregiver needs to reach an elder.
// The real Kakao Mobility / TMAP client arrives in stage 5; until then
// Schedule stands in for it.
package eta

import (
	"context"
	"math"
	"time"
)

// Request is one caregiver position for a visit.
type Request struct {
	Latitude      float64
	Longitude     float64
	RecordedAt    time.Time
	ScheduledTime time.Time
}

// Estimator returns the caregiver's remaining travel time in whole minutes.
type Estimator interface {
	Minutes(ctx context.Context, r Request) (int, error)
}

// Schedule is a fake Estimator: the caregiver arrives exactly at the visit's
// scheduled time, so the ETA is the time left until then (never negative).
type Schedule struct {
	Now func() time.Time
}

// Minutes implements Estimator.
func (s Schedule) Minutes(_ context.Context, r Request) (int, error) {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	left := r.ScheduledTime.Sub(now()).Minutes()
	if left <= 0 {
		return 0, nil
	}
	return int(math.Ceil(left)), nil
}
