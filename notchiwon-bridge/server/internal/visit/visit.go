// Package visit handles caregiver visits: the day's list, location updates
// that refresh the ETA and start the pickup session, and arrival.
package visit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/eta"
)

var (
	// ErrNotFound means the visit does not exist or belongs to another caregiver.
	ErrNotFound = errors.New("visit not found")
	// ErrClosed means the visit is already completed or cancelled.
	ErrClosed = errors.New("visit closed")
)

// Status values of visit.status.
const (
	StatusScheduled     = "SCHEDULED"
	StatusEnRoute       = "EN_ROUTE"
	StatusSessionActive = "SESSION_ACTIVE"
	StatusCompleted     = "COMPLETED"
	StatusCancelled     = "CANCELLED"
)

// SessionNotifier hears about sessions this package starts or ends, after the
// transaction commits. The /ws/elder handler implements it to tell the tablet.
type SessionNotifier interface {
	SessionStarted(ctx context.Context, s db.ConversationSession, etaMinutes int)
	SessionEnded(ctx context.Context, s db.ConversationSession)
	// EtaUpdated reports a fresh ETA for the visit of an open pickup session.
	EtaUpdated(ctx context.Context, s db.ConversationSession, etaMinutes int)
}

// Notifiers fans one event out to several SessionNotifiers, in order.
type Notifiers []SessionNotifier

// SessionStarted implements SessionNotifier.
func (n Notifiers) SessionStarted(ctx context.Context, s db.ConversationSession, etaMinutes int) {
	for _, x := range n {
		x.SessionStarted(ctx, s, etaMinutes)
	}
}

// SessionEnded implements SessionNotifier.
func (n Notifiers) SessionEnded(ctx context.Context, s db.ConversationSession) {
	for _, x := range n {
		x.SessionEnded(ctx, s)
	}
}

// EtaUpdated implements SessionNotifier.
func (n Notifiers) EtaUpdated(ctx context.Context, s db.ConversationSession, etaMinutes int) {
	for _, x := range n {
		x.EtaUpdated(ctx, s, etaMinutes)
	}
}

// Detail is a visit with the fields the caregiver app shows.
type Detail struct {
	Visit     db.Visit
	ElderName string
	SessionID *uuid.UUID
	// Home is the elder's home, nil when it is not registered.
	Home *Home
}

// Home is where the caregiver drives to.
type Home struct {
	Address   *string
	Latitude  float64
	Longitude float64
}

func home(address *string, lat, lng *float64) *Home {
	if lat == nil || lng == nil {
		return nil
	}
	return &Home{Address: address, Latitude: *lat, Longitude: *lng}
}

// Location is one caregiver position.
type Location struct {
	Latitude   float64
	Longitude  float64
	AccuracyM  *float64
	RecordedAt time.Time
}

// LocationResult is what a location update did.
type LocationResult struct {
	Status         string
	EtaMinutes     int
	SessionStarted bool
	SessionID      *uuid.UUID
}

// Config is the Service's settings.
type Config struct {
	// TriggerEtaMinutes starts the pickup session once the ETA is at or below it.
	TriggerEtaMinutes int
	// Location defines "today".
	Location *time.Location
	// PromptVersion is recorded on the sessions this package starts.
	PromptVersion string
}

// Service implements the visit use cases.
type Service struct {
	pool     *pgxpool.Pool
	eta      eta.Estimator
	notifier SessionNotifier
	cfg      Config
	logger   *slog.Logger
	now      func() time.Time
}

// NewService returns a Service. notifier may be nil.
func NewService(pool *pgxpool.Pool, estimator eta.Estimator, notifier SessionNotifier, cfg Config, logger *slog.Logger) *Service {
	if notifier == nil {
		notifier = nopNotifier{}
	}
	if cfg.Location == nil {
		cfg.Location = time.UTC
	}
	return &Service{pool: pool, eta: estimator, notifier: notifier, cfg: cfg, logger: logger, now: time.Now}
}

// ListToday returns the caregiver's visits scheduled today (in cfg.Location), earliest first.
func (s *Service) ListToday(ctx context.Context, caregiverID uuid.UUID) ([]Detail, error) {
	now := s.now().In(s.cfg.Location)
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.cfg.Location)
	rows, err := db.New(s.pool).ListCaregiverVisitsBetween(ctx, db.ListCaregiverVisitsBetweenParams{
		CaregiverID: caregiverID,
		FromTime:    from,
		ToTime:      from.AddDate(0, 0, 1),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Detail, len(rows))
	for i, r := range rows {
		out[i] = Detail{Visit: r.Visit, ElderName: r.ElderName, SessionID: r.SessionID,
			Home: home(r.HomeAddress, r.HomeLatitude, r.HomeLongitude)}
	}
	return out, nil
}

// Get returns the caregiver's visit.
func (s *Service) Get(ctx context.Context, caregiverID, visitID uuid.UUID) (Detail, error) {
	return get(ctx, db.New(s.pool), caregiverID, visitID)
}

func get(ctx context.Context, q *db.Queries, caregiverID, visitID uuid.UUID) (Detail, error) {
	r, err := q.GetVisitDetail(ctx, visitID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && r.Visit.CaregiverID != caregiverID) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	return Detail{Visit: r.Visit, ElderName: r.ElderName, SessionID: r.SessionID,
		Home: home(r.HomeAddress, r.HomeLatitude, r.HomeLongitude)}, nil
}

func closed(status string) bool {
	return status == StatusCompleted || status == StatusCancelled
}

// RecordLocation stores the caregiver's position, refreshes the visit ETA and,
// once the ETA reaches the trigger, starts the pickup session.
//
// Concurrent calls for one visit serialize on the visit row (the ETA UPDATE
// takes its lock), and ClaimVisitSession only succeeds while the visit is
// SCHEDULED or EN_ROUTE, so exactly one call starts the session.
func (s *Service) RecordLocation(ctx context.Context, caregiverID, visitID uuid.UUID, loc Location) (LocationResult, error) {
	d, err := s.Get(ctx, caregiverID, visitID)
	if err != nil {
		return LocationResult{}, err
	}
	if closed(d.Visit.Status) {
		return LocationResult{}, ErrClosed
	}

	req := eta.Request{
		Latitude: loc.Latitude, Longitude: loc.Longitude,
		RecordedAt: loc.RecordedAt, ScheduledTime: d.Visit.ScheduledTime,
	}
	if d.Home != nil {
		req.Destination = &eta.Point{Latitude: d.Home.Latitude, Longitude: d.Home.Longitude}
	}
	minutes, err := s.eta.Minutes(ctx, req)
	if err != nil {
		return LocationResult{}, fmt.Errorf("estimate eta: %w", err)
	}
	etaAt := s.now().Add(time.Duration(minutes) * time.Minute)

	var (
		res       = LocationResult{EtaMinutes: minutes}
		started   *db.ConversationSession
		preempted *db.ConversationSession
		active    *db.ConversationSession
	)
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		m := int32(minutes)
		if err := q.CreateCaregiverLocation(ctx, db.CreateCaregiverLocationParams{
			CaregiverID: caregiverID, VisitID: &visitID,
			Latitude: loc.Latitude, Longitude: loc.Longitude, AccuracyM: loc.AccuracyM,
			EtaMinutes: &m, RecordedAt: loc.RecordedAt,
		}); err != nil {
			return fmt.Errorf("save location: %w", err)
		}

		v, err := q.RecordVisitEta(ctx, db.RecordVisitEtaParams{ID: visitID, EtaCurrent: &etaAt})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrClosed // completed or cancelled since Get
		}
		if err != nil {
			return fmt.Errorf("record eta: %w", err)
		}
		res.Status = v.Status

		if minutes <= s.cfg.TriggerEtaMinutes && v.Status != StatusSessionActive {
			sess, prev, err := s.startPickupSession(ctx, q, visitID, m)
			if err != nil {
				return err
			}
			if sess != nil {
				started, preempted = sess, prev
				res.Status = StatusSessionActive
				res.SessionStarted = true
				res.SessionID = &sess.ID
				return nil
			}
		}

		if res.Status == StatusSessionActive {
			sess, err := q.GetSessionByVisit(ctx, &visitID)
			if err != nil {
				return fmt.Errorf("read session: %w", err)
			}
			res.SessionID = &sess.ID
			if sess.EndedAt == nil {
				active = &sess
			}
		}
		return nil
	})
	if err != nil {
		return LocationResult{}, err
	}

	if preempted != nil {
		s.notifier.SessionEnded(ctx, *preempted)
	}
	if started != nil {
		s.logger.InfoContext(ctx, "pickup session started",
			"visit_id", visitID, "session_id", started.ID, "eta_minutes", minutes)
		s.notifier.SessionStarted(ctx, *started, minutes)
	}
	if active != nil {
		s.notifier.EtaUpdated(ctx, *active, minutes)
	}
	return res, nil
}

// startPickupSession claims the visit and creates its session. It returns a nil
// session when another request already claimed it. A companion session still
// open for the elder is ended as PREEMPTED and returned as prev.
func (s *Service) startPickupSession(ctx context.Context, q *db.Queries, visitID uuid.UUID, etaMinutes int32) (sess, prev *db.ConversationSession, err error) {
	v, err := q.ClaimVisitSession(ctx, visitID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("claim visit: %w", err)
	}
	p, err := q.PreemptCompanionSession(ctx, v.ElderID)
	switch {
	case err == nil:
		prev = &p
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, nil, fmt.Errorf("preempt companion session: %w", err)
	}
	created, err := q.CreatePickupSession(ctx, db.CreatePickupSessionParams{
		VisitID: &v.ID, ElderID: v.ElderID, TriggerEtaMinutes: &etaMinutes,
		PromptVersion: nullable(s.cfg.PromptVersion),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}
	return &created, prev, nil
}

// Arrive completes the visit and ends its session as CAREGIVER_ARRIVED.
func (s *Service) Arrive(ctx context.Context, caregiverID, visitID uuid.UUID) (Detail, error) {
	if _, err := s.Get(ctx, caregiverID, visitID); err != nil {
		return Detail{}, err
	}
	var (
		d     Detail
		ended *db.ConversationSession
	)
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		_, err := q.CompleteVisit(ctx, db.CompleteVisitParams{ID: visitID, ArrivedAt: ptr(s.now())})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrClosed
		}
		if err != nil {
			return fmt.Errorf("complete visit: %w", err)
		}
		sess, err := q.GetSessionByVisit(ctx, &visitID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return fmt.Errorf("read session: %w", err)
		case sess.EndedAt == nil:
			e, err := q.EndSession(ctx, db.EndSessionParams{ID: sess.ID, EndedReason: ptr("CAREGIVER_ARRIVED")})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("end session: %w", err)
			}
			if err == nil {
				ended = &e
			}
		}
		d, err = get(ctx, q, caregiverID, visitID)
		return err
	})
	if err != nil {
		return Detail{}, err
	}
	if ended != nil {
		s.notifier.SessionEnded(ctx, *ended)
	}
	return d, nil
}

func ptr[T any](v T) *T { return &v }

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type nopNotifier struct{}

func (nopNotifier) SessionStarted(context.Context, db.ConversationSession, int) {}
func (nopNotifier) SessionEnded(context.Context, db.ConversationSession)        {}
func (nopNotifier) EtaUpdated(context.Context, db.ConversationSession, int)     {}
