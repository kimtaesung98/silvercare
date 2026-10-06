package httpapi

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/apigen"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/visit"
)

// Server implements apigen.StrictServerInterface.
type Server struct {
	deps Deps
	q    *db.Queries
}

var _ apigen.StrictServerInterface = (*Server)(nil)

func apiError(code, message string) apigen.ApiError {
	return apigen.ApiError{Code: code, Message: message}
}

var (
	errNotFound     = apiError("NOT_FOUND", "찾을 수 없습니다.")
	errUnauthorized = apiError("UNAUTHENTICATED", "아이디 또는 비밀번호가 맞지 않습니다.")
	errVisitClosed  = apiError("VISIT_CLOSED", "이미 완료되었거나 취소된 방문입니다.")
)

func invalid(message string) apigen.ApiError { return apiError("INVALID_ARGUMENT", message) }

// mustCaregiver returns the caregiver authenticate stored. Its absence is a wiring bug.
func mustCaregiver(ctx context.Context) (auth.Caregiver, error) {
	c, ok := auth.CaregiverFrom(ctx)
	if !ok {
		return auth.Caregiver{}, errors.New("no caregiver in context")
	}
	return c, nil
}

func toVisit(d visit.Detail) apigen.Visit {
	return apigen.Visit{
		Id:                d.Visit.ID,
		Elder:             apigen.ElderSummary{Id: d.Visit.ElderID, Name: d.ElderName},
		ScheduledTime:     d.Visit.ScheduledTime,
		EtaCurrent:        d.Visit.EtaCurrent,
		Status:            apigen.VisitStatus(d.Visit.Status),
		ActualArrivalTime: d.Visit.ActualArrivalTime,
		SessionId:         d.SessionID,
	}
}

// GetHealth implements GET /healthz.
func (s *Server) GetHealth(context.Context, apigen.GetHealthRequestObject) (apigen.GetHealthResponseObject, error) {
	return apigen.GetHealth200JSONResponse{Status: "ok"}, nil
}

// LoginCaregiver implements POST /auth/caregiver/login (temporary center-issued accounts).
func (s *Server) LoginCaregiver(ctx context.Context, req apigen.LoginCaregiverRequestObject) (apigen.LoginCaregiverResponseObject, error) {
	if req.Body == nil {
		return apigen.LoginCaregiver401JSONResponse{UnauthorizedJSONResponse: apigen.UnauthorizedJSONResponse(errUnauthorized)}, nil
	}
	c, err := s.q.GetCaregiverByLoginID(ctx, &req.Body.LoginId)
	var hash *string
	switch {
	case err == nil:
		hash = c.PasswordHash
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, err
	}
	if !auth.CheckPasswordOrDummy(hash, req.Body.Password) {
		return apigen.LoginCaregiver401JSONResponse{UnauthorizedJSONResponse: apigen.UnauthorizedJSONResponse(errUnauthorized)}, nil
	}
	tok, exp := s.deps.Tokens.IssueCaregiver(c.ID)
	return apigen.LoginCaregiver200JSONResponse{
		AccessToken: tok,
		ExpiresAt:   exp,
		Caregiver:   apigen.Caregiver{Id: c.ID, Name: c.Name, CenterId: c.CenterID},
	}, nil
}

// ListTodayVisits implements GET /visits/today.
func (s *Server) ListTodayVisits(ctx context.Context, _ apigen.ListTodayVisitsRequestObject) (apigen.ListTodayVisitsResponseObject, error) {
	c, err := mustCaregiver(ctx)
	if err != nil {
		return nil, err
	}
	ds, err := s.deps.Visits.ListToday(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	out := apigen.ListTodayVisits200JSONResponse{Visits: make([]apigen.Visit, len(ds))}
	for i, d := range ds {
		out.Visits[i] = toVisit(d)
	}
	return out, nil
}

// GetVisit implements GET /visits/{visitId}.
func (s *Server) GetVisit(ctx context.Context, req apigen.GetVisitRequestObject) (apigen.GetVisitResponseObject, error) {
	c, err := mustCaregiver(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.deps.Visits.Get(ctx, c.ID, req.VisitId)
	if errors.Is(err, visit.ErrNotFound) {
		return apigen.GetVisit404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}, nil
	}
	if err != nil {
		return nil, err
	}
	return apigen.GetVisit200JSONResponse(toVisit(d)), nil
}

func validateLocation(b *apigen.LocationUpdate) error {
	switch {
	case b == nil:
		return errors.New("요청 본문이 필요합니다.")
	case math.IsNaN(b.Latitude) || b.Latitude < -90 || b.Latitude > 90:
		return errors.New("latitude는 -90~90이어야 합니다.")
	case math.IsNaN(b.Longitude) || b.Longitude < -180 || b.Longitude > 180:
		return errors.New("longitude는 -180~180이어야 합니다.")
	case b.AccuracyM != nil && (math.IsNaN(*b.AccuracyM) || *b.AccuracyM < 0):
		return errors.New("accuracyM은 0 이상이어야 합니다.")
	case b.RecordedAt.IsZero():
		return errors.New("recordedAt이 필요합니다.")
	}
	return nil
}

// PostVisitLocation implements POST /visits/{visitId}/location.
func (s *Server) PostVisitLocation(ctx context.Context, req apigen.PostVisitLocationRequestObject) (apigen.PostVisitLocationResponseObject, error) {
	c, err := mustCaregiver(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateLocation(req.Body); err != nil {
		return apigen.PostVisitLocation400JSONResponse{BadRequestJSONResponse: apigen.BadRequestJSONResponse(invalid(err.Error()))}, nil
	}
	res, err := s.deps.Visits.RecordLocation(ctx, c.ID, req.VisitId, visit.Location{
		Latitude: req.Body.Latitude, Longitude: req.Body.Longitude,
		AccuracyM: req.Body.AccuracyM, RecordedAt: req.Body.RecordedAt,
	})
	switch {
	case errors.Is(err, visit.ErrNotFound):
		return apigen.PostVisitLocation404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}, nil
	case errors.Is(err, visit.ErrClosed):
		return apigen.PostVisitLocation409JSONResponse(errVisitClosed), nil
	case err != nil:
		return nil, err
	}
	return apigen.PostVisitLocation200JSONResponse{
		VisitId:        req.VisitId,
		Status:         apigen.VisitStatus(res.Status),
		EtaMinutes:     int32(res.EtaMinutes),
		SessionStarted: res.SessionStarted,
		SessionId:      res.SessionID,
	}, nil
}

// ArriveVisit implements POST /visits/{visitId}/arrive.
func (s *Server) ArriveVisit(ctx context.Context, req apigen.ArriveVisitRequestObject) (apigen.ArriveVisitResponseObject, error) {
	c, err := mustCaregiver(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.deps.Visits.Arrive(ctx, c.ID, req.VisitId)
	switch {
	case errors.Is(err, visit.ErrNotFound):
		return apigen.ArriveVisit404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}, nil
	case errors.Is(err, visit.ErrClosed):
		return apigen.ArriveVisit409JSONResponse(errVisitClosed), nil
	case err != nil:
		return nil, err
	}
	return apigen.ArriveVisit200JSONResponse(toVisit(d)), nil
}

// PutCaregiverFcmToken implements PUT /devices/me/fcm-token.
func (s *Server) PutCaregiverFcmToken(ctx context.Context, req apigen.PutCaregiverFcmTokenRequestObject) (apigen.PutCaregiverFcmTokenResponseObject, error) {
	c, err := mustCaregiver(ctx)
	if err != nil {
		return nil, err
	}
	if req.Body == nil || strings.TrimSpace(req.Body.FcmToken) == "" {
		return apigen.PutCaregiverFcmToken400JSONResponse{BadRequestJSONResponse: apigen.BadRequestJSONResponse(invalid("fcmToken이 필요합니다."))}, nil
	}
	if _, err := s.q.RegisterCaregiverPhone(ctx, db.RegisterCaregiverPhoneParams{
		CaregiverID: &c.ID, FcmToken: &req.Body.FcmToken, Label: req.Body.Label,
	}); err != nil {
		return nil, fmt.Errorf("register phone: %w", err)
	}
	return apigen.PutCaregiverFcmToken204Response{}, nil
}

// GetTabletContext implements GET /tablet/me.
func (s *Server) GetTabletContext(ctx context.Context, _ apigen.GetTabletContextRequestObject) (apigen.GetTabletContextResponseObject, error) {
	t, ok := auth.TabletFrom(ctx)
	if !ok {
		return nil, errors.New("no tablet in context")
	}
	elder, err := s.q.GetElder(ctx, t.ElderID)
	if err != nil {
		return nil, fmt.Errorf("read elder: %w", err)
	}
	out := apigen.GetTabletContext200JSONResponse{
		DeviceId:          t.DeviceID,
		Elder:             apigen.ElderSummary{Id: elder.ID, Name: elder.Name},
		PreferredTtsVoice: elder.PreferredTtsVoice,
	}
	sched, err := s.q.GetCompanionSchedule(ctx, elder.ID)
	switch {
	case err == nil:
		out.CompanionEnabled = sched.Enabled
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("read companion schedule: %w", err)
	}
	sess, err := s.q.GetOpenSessionForElder(ctx, elder.ID)
	switch {
	case err == nil:
		out.ActiveSession = &apigen.ActiveSession{Id: sess.ID, Mode: apigen.SessionMode(sess.Mode), StartedAt: sess.StartedAt}
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("read open session: %w", err)
	}
	return out, nil
}

// The endpoints below belong to later stages (development-process.md) and answer 501 until then.

// GetSessionBriefing is stage 5.
func (s *Server) GetSessionBriefing(context.Context, apigen.GetSessionBriefingRequestObject) (apigen.GetSessionBriefingResponseObject, error) {
	return nil, errNotImplemented
}

// MarkBriefingRead is stage 5.
func (s *Server) MarkBriefingRead(context.Context, apigen.MarkBriefingReadRequestObject) (apigen.MarkBriefingReadResponseObject, error) {
	return nil, errNotImplemented
}

// GetEscalation is stage 5.
func (s *Server) GetEscalation(context.Context, apigen.GetEscalationRequestObject) (apigen.GetEscalationResponseObject, error) {
	return nil, errNotImplemented
}

// AckEscalation is stage 5.
func (s *Server) AckEscalation(context.Context, apigen.AckEscalationRequestObject) (apigen.AckEscalationResponseObject, error) {
	return nil, errNotImplemented
}

// ListOpenerClips is stage 3 (seed data) and 4 (audio cache).
func (s *Server) ListOpenerClips(context.Context, apigen.ListOpenerClipsRequestObject) (apigen.ListOpenerClipsResponseObject, error) {
	return nil, errNotImplemented
}

// GetOpenerClipAudio is stage 4.
func (s *Server) GetOpenerClipAudio(context.Context, apigen.GetOpenerClipAudioRequestObject) (apigen.GetOpenerClipAudioResponseObject, error) {
	return nil, errNotImplemented
}

// GetCompanionSchedule is stage 6.
func (s *Server) GetCompanionSchedule(context.Context, apigen.GetCompanionScheduleRequestObject) (apigen.GetCompanionScheduleResponseObject, error) {
	return nil, errNotImplemented
}

// PutCompanionSchedule is stage 6.
func (s *Server) PutCompanionSchedule(context.Context, apigen.PutCompanionScheduleRequestObject) (apigen.PutCompanionScheduleResponseObject, error) {
	return nil, errNotImplemented
}

// GetDailyDigest is stage 6.
func (s *Server) GetDailyDigest(context.Context, apigen.GetDailyDigestRequestObject) (apigen.GetDailyDigestResponseObject, error) {
	return nil, errNotImplemented
}
