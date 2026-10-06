package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/apigen"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/opener"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/speech"
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
	v := apigen.Visit{
		Id:                d.Visit.ID,
		Elder:             apigen.ElderSummary{Id: d.Visit.ElderID, Name: d.ElderName},
		ScheduledTime:     d.Visit.ScheduledTime,
		EtaCurrent:        d.Visit.EtaCurrent,
		Status:            apigen.VisitStatus(d.Visit.Status),
		ActualArrivalTime: d.Visit.ActualArrivalTime,
		SessionId:         d.SessionID,
	}
	if h := d.Home; h != nil {
		v.Destination = &apigen.Place{Latitude: h.Latitude, Longitude: h.Longitude, Address: h.Address}
	}
	return v
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

// errBriefingNotReady answers while the briefing job has not run yet; the
// app retries.
var errBriefingNotReady = apiError("BRIEFING_NOT_READY", "브리핑을 만들고 있습니다. 잠시 후 다시 시도해 주세요.")

// ownSession reports whether the session belongs to one of the caregiver's
// visits. Companion sessions belong to no caregiver.
func (s *Server) ownSession(ctx context.Context, caregiverID, sessionID uuid.UUID) (bool, error) {
	o, err := s.q.GetSessionOwner(ctx, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read session: %w", err)
	}
	return o.CaregiverID != nil && *o.CaregiverID == caregiverID, nil
}

func (s *Server) briefing(ctx context.Context, sessionID uuid.UUID) (apigen.Briefing, error) {
	d, err := s.q.GetBriefingDetail(ctx, sessionID)
	if err != nil {
		return apigen.Briefing{}, err
	}
	return toBriefing(d), nil
}

func toBriefing(d db.GetBriefingDetailRow) apigen.Briefing {
	b := d.BriefingReport
	out := apigen.Briefing{
		SessionId:         b.SessionID,
		SummaryText:       b.SummaryText,
		TopKeywords:       []string{},
		EmotionFlag:       b.EmotionFlag,
		EscalationCount:   &d.EscalationCount,
		GeneratedAt:       b.GeneratedAt,
		ReadByCaregiverAt: b.ReadByCaregiverAt,
	}
	_ = json.Unmarshal(b.TopKeywords, &out.TopKeywords)
	if d.OverallEmotionTag != nil {
		tag := apigen.EmotionTag(*d.OverallEmotionTag)
		out.OverallEmotionTag = &tag
	}
	return out
}

// GetSessionBriefing implements GET /sessions/{sessionId}/briefing.
func (s *Server) GetSessionBriefing(ctx context.Context, req apigen.GetSessionBriefingRequestObject) (apigen.GetSessionBriefingResponseObject, error) {
	c, err := mustCaregiver(ctx)
	if err != nil {
		return nil, err
	}
	if ok, err := s.ownSession(ctx, c.ID, req.SessionId); err != nil {
		return nil, err
	} else if !ok {
		return apigen.GetSessionBriefing404JSONResponse(errNotFound), nil
	}
	b, err := s.briefing(ctx, req.SessionId)
	if errors.Is(err, pgx.ErrNoRows) {
		return apigen.GetSessionBriefing404JSONResponse(errBriefingNotReady), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read briefing: %w", err)
	}
	return apigen.GetSessionBriefing200JSONResponse(b), nil
}

// MarkBriefingRead implements POST /sessions/{sessionId}/briefing/read.
func (s *Server) MarkBriefingRead(ctx context.Context, req apigen.MarkBriefingReadRequestObject) (apigen.MarkBriefingReadResponseObject, error) {
	c, err := mustCaregiver(ctx)
	if err != nil {
		return nil, err
	}
	notFound := apigen.MarkBriefingRead404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}
	if ok, err := s.ownSession(ctx, c.ID, req.SessionId); err != nil {
		return nil, err
	} else if !ok {
		return notFound, nil
	}
	if _, err := s.q.MarkBriefingRead(ctx, req.SessionId); errors.Is(err, pgx.ErrNoRows) {
		return notFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("mark briefing read: %w", err)
	}
	b, err := s.briefing(ctx, req.SessionId)
	if err != nil {
		return nil, fmt.Errorf("read briefing: %w", err)
	}
	return apigen.MarkBriefingRead200JSONResponse(b), nil
}

// escalationRow is the columns GetEscalationDetail and
// ListOpenEscalationsForCaregiver share.
type escalationRow struct {
	ev            db.EscalationEvent
	elderID       uuid.UUID
	elderName     string
	utteranceText *string
	visitID       *uuid.UUID
}

func toEscalation(r escalationRow) apigen.Escalation {
	return apigen.Escalation{
		Id:             r.ev.ID,
		SessionId:      r.ev.SessionID,
		Elder:          apigen.ElderSummary{Id: r.elderID, Name: r.elderName},
		TriggerType:    apigen.EscalationTriggerType(r.ev.TriggerType),
		Source:         apigen.EscalationSource(r.ev.Source),
		UtteranceText:  r.utteranceText,
		Reason:         r.ev.Reason,
		CreatedAt:      r.ev.CreatedAt,
		AcknowledgedAt: r.ev.AcknowledgedAt,
		VisitId:        r.visitID,
	}
}

// escalationFor reads an escalation the caregiver may see (one raised during
// their own visit), or reports false.
func (s *Server) escalationFor(ctx context.Context, caregiverID, id uuid.UUID) (escalationRow, bool, error) {
	d, err := s.q.GetEscalationDetail(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return escalationRow{}, false, nil
	}
	if err != nil {
		return escalationRow{}, false, fmt.Errorf("read escalation: %w", err)
	}
	if d.CaregiverID == nil || *d.CaregiverID != caregiverID {
		return escalationRow{}, false, nil
	}
	return escalationRow{d.EscalationEvent, d.ElderID, d.ElderName, d.UtteranceText, d.VisitID}, true, nil
}

// ListOpenEscalations implements GET /escalations/open.
func (s *Server) ListOpenEscalations(ctx context.Context, _ apigen.ListOpenEscalationsRequestObject) (apigen.ListOpenEscalationsResponseObject, error) {
	c, err := mustCaregiver(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListOpenEscalationsForCaregiver(ctx, db.ListOpenEscalationsForCaregiverParams{
		CaregiverID: c.ID, Since: time.Now().Add(-openEscalationWindow),
	})
	if err != nil {
		return nil, fmt.Errorf("list open escalations: %w", err)
	}
	out := apigen.ListOpenEscalations200JSONResponse{Escalations: make([]apigen.Escalation, len(rows))}
	for i, r := range rows {
		out.Escalations[i] = toEscalation(escalationRow{r.EscalationEvent, r.ElderID, r.ElderName, r.UtteranceText, &r.VisitID})
	}
	return out, nil
}

// openEscalationWindow is how far back GET /escalations/open looks.
const openEscalationWindow = 24 * time.Hour

// GetEscalation implements GET /escalations/{escalationId}.
func (s *Server) GetEscalation(ctx context.Context, req apigen.GetEscalationRequestObject) (apigen.GetEscalationResponseObject, error) {
	c, err := mustCaregiver(ctx)
	if err != nil {
		return nil, err
	}
	r, ok, err := s.escalationFor(ctx, c.ID, req.EscalationId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return apigen.GetEscalation404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}, nil
	}
	return apigen.GetEscalation200JSONResponse(toEscalation(r)), nil
}

// AckEscalation implements POST /escalations/{escalationId}/ack.
func (s *Server) AckEscalation(ctx context.Context, req apigen.AckEscalationRequestObject) (apigen.AckEscalationResponseObject, error) {
	c, err := mustCaregiver(ctx)
	if err != nil {
		return nil, err
	}
	r, ok, err := s.escalationFor(ctx, c.ID, req.EscalationId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return apigen.AckEscalation404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}, nil
	}
	ev, err := s.q.AcknowledgeEscalation(ctx, db.AcknowledgeEscalationParams{ID: req.EscalationId, CaregiverID: &c.ID})
	if err != nil {
		return nil, fmt.Errorf("acknowledge escalation: %w", err)
	}
	r.ev = ev
	return apigen.AckEscalation200JSONResponse(toEscalation(r)), nil
}

// ListOpenerClips implements GET /tablet/opener-clips.
func (s *Server) ListOpenerClips(ctx context.Context, _ apigen.ListOpenerClipsRequestObject) (apigen.ListOpenerClipsResponseObject, error) {
	t, ok := auth.TabletFrom(ctx)
	if !ok {
		return nil, errors.New("no tablet in context")
	}
	elder, err := s.q.GetElder(ctx, t.ElderID)
	if err != nil {
		return nil, fmt.Errorf("read elder: %w", err)
	}
	m, err := s.deps.Openers.Manifest(ctx, opener.VoiceOf(elder))
	if err != nil {
		return nil, err
	}
	out := apigen.ListOpenerClips200JSONResponse{Voice: m.Voice, Version: m.Version, Clips: make([]apigen.OpenerClip, len(m.Clips))}
	for i, c := range m.Clips {
		out.Clips[i] = apigen.OpenerClip{
			Id: c.ID, Category: apigen.OpenerCategory(c.Category), Text: c.Text, DurationMs: c.DurationMs,
			AudioUrl: "/tablet/opener-clips/" + c.ID.String() + "/audio",
		}
	}
	return out, nil
}

// GetOpenerClipAudio implements GET /tablet/opener-clips/{clipId}/audio.
// A clip without audio yet is synthesized with Clova TTS on first request.
func (s *Server) GetOpenerClipAudio(ctx context.Context, req apigen.GetOpenerClipAudioRequestObject) (apigen.GetOpenerClipAudioResponseObject, error) {
	mp3, err := s.deps.Openers.Audio(ctx, req.ClipId, s.deps.TTS)
	switch {
	case errors.Is(err, opener.ErrNoClip):
		return apigen.GetOpenerClipAudio404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}, nil
	case errors.Is(err, speech.ErrUnavailable):
		return apigen.GetOpenerClipAudio404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(
			apiError("AUDIO_NOT_READY", "음성이 아직 준비되지 않았습니다."))}, nil
	case err != nil:
		return nil, err
	}
	return apigen.GetOpenerClipAudio200AudiompegResponse{Body: bytes.NewReader(mp3), ContentLength: int64(len(mp3))}, nil
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
