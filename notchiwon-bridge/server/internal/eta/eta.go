// Package eta estimates how many minutes a caregiver needs to reach an elder.
// Kakao is the real estimator (Kakao Mobility directions); Schedule stands in
// when no API key is set or the elder's home is not registered.
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
	// Destination is the elder's home, nil when it is not registered.
	Destination *Point
}

// Point is a WGS84 coordinate.
type Point struct {
	Latitude  float64
	Longitude float64
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
	return ceilMinutes(r.ScheduledTime.Sub(now())), nil
}

// ceilMinutes rounds d up to whole minutes, never below zero.
func ceilMinutes(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int(math.Ceil(d.Minutes()))
}
