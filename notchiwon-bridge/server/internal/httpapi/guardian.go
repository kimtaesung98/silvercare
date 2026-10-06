package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/apigen"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
)

// mustGuardian returns the guardian authenticate stored. Its absence is a
// wiring bug.
func mustGuardian(ctx context.Context) (auth.Guardian, error) {
	g, ok := auth.GuardianFrom(ctx)
	if !ok {
		return auth.Guardian{}, errors.New("no guardian in context")
	}
	return g, nil
}

// LoginGuardian implements POST /auth/guardian/login (temporary center-issued accounts).
func (s *Server) LoginGuardian(ctx context.Context, req apigen.LoginGuardianRequestObject) (apigen.LoginGuardianResponseObject, error) {
	unauthorized := apigen.LoginGuardian401JSONResponse{UnauthorizedJSONResponse: apigen.UnauthorizedJSONResponse(errUnauthorized)}
	if req.Body == nil {
		return unauthorized, nil
	}
	g, err := s.q.GetGuardianByLoginID(ctx, &req.Body.LoginId)
	var hash *string
	switch {
	case err == nil:
		hash = g.PasswordHash
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, err
	}
	if !auth.CheckPasswordOrDummy(hash, req.Body.Password) {
		return unauthorized, nil
	}
	elders, err := s.elders(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	tok, exp := s.deps.Tokens.IssueGuardian(g.ID)
	return apigen.LoginGuardian200JSONResponse{
		AccessToken: tok,
		ExpiresAt:   exp,
		Guardian:    apigen.Guardian{Id: g.ID, Name: g.Name},
		Elders:      elders,
	}, nil
}

// GetGuardianContext implements GET /guardian/me.
func (s *Server) GetGuardianContext(ctx context.Context, _ apigen.GetGuardianContextRequestObject) (apigen.GetGuardianContextResponseObject, error) {
	g, err := mustGuardian(ctx)
	if err != nil {
		return nil, err
	}
	row, err := s.q.GetGuardian(ctx, g.ID)
	if err != nil {
		return nil, fmt.Errorf("read guardian: %w", err)
	}
	elders, err := s.elders(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	return apigen.GetGuardianContext200JSONResponse{
		Guardian: apigen.Guardian{Id: row.ID, Name: row.Name},
		Elders:   elders,
	}, nil
}

func (s *Server) elders(ctx context.Context, guardianID uuid.UUID) ([]apigen.ElderSummary, error) {
	rows, err := s.q.ListGuardianElders(ctx, guardianID)
	if err != nil {
		return nil, fmt.Errorf("list elders: %w", err)
	}
	out := make([]apigen.ElderSummary, len(rows))
	for i, r := range rows {
		out[i] = apigen.ElderSummary{Id: r.ID, Name: r.Name}
	}
	return out, nil
}

// myWard reports whether the elder is one the guardian cares for.
func (s *Server) myWard(ctx context.Context, elderID uuid.UUID) (bool, error) {
	g, err := mustGuardian(ctx)
	if err != nil {
		return false, err
	}
	elder, err := s.q.GetElder(ctx, elderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read elder: %w", err)
	}
	return elder.GuardianID == g.ID, nil
}

// PutGuardianFcmToken implements PUT /guardian/devices/me/fcm-token.
func (s *Server) PutGuardianFcmToken(ctx context.Context, req apigen.PutGuardianFcmTokenRequestObject) (apigen.PutGuardianFcmTokenResponseObject, error) {
	g, err := mustGuardian(ctx)
	if err != nil {
		return nil, err
	}
	if req.Body == nil || strings.TrimSpace(req.Body.FcmToken) == "" {
		return apigen.PutGuardianFcmToken400JSONResponse{
			BadRequestJSONResponse: apigen.BadRequestJSONResponse(invalid("fcmToken이 필요합니다.")),
		}, nil
	}
	if _, err := s.q.RegisterGuardianPhone(ctx, db.RegisterGuardianPhoneParams{
		GuardianID: &g.ID, FcmToken: &req.Body.FcmToken, Label: req.Body.Label,
	}); err != nil {
		return nil, fmt.Errorf("register phone: %w", err)
	}
	return apigen.PutGuardianFcmToken204Response{}, nil
}

// GetGuardianCompanionSchedule implements GET /guardian/elders/{elderId}/companion-schedule.
func (s *Server) GetGuardianCompanionSchedule(ctx context.Context, req apigen.GetGuardianCompanionScheduleRequestObject) (apigen.GetGuardianCompanionScheduleResponseObject, error) {
	if ok, err := s.myWard(ctx, req.ElderId); err != nil {
		return nil, err
	} else if !ok {
		return apigen.GetGuardianCompanionSchedule404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}, nil
	}
	sched, err := s.schedule(ctx, req.ElderId)
	if err != nil {
		return nil, err
	}
	return apigen.GetGuardianCompanionSchedule200JSONResponse(sched), nil
}

// PutGuardianCompanionSchedule implements PUT /guardian/elders/{elderId}/companion-schedule.
func (s *Server) PutGuardianCompanionSchedule(ctx context.Context, req apigen.PutGuardianCompanionScheduleRequestObject) (apigen.PutGuardianCompanionScheduleResponseObject, error) {
	if ok, err := s.myWard(ctx, req.ElderId); err != nil {
		return nil, err
	} else if !ok {
		return apigen.PutGuardianCompanionSchedule404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}, nil
	}
	if req.Body == nil {
		return apigen.PutGuardianCompanionSchedule400JSONResponse{
			BadRequestJSONResponse: apigen.BadRequestJSONResponse(invalid("설정이 필요합니다.")),
		}, nil
	}
	saved, bad := s.saveSchedule(ctx, req.ElderId, *req.Body)
	if bad != nil {
		return apigen.PutGuardianCompanionSchedule400JSONResponse{BadRequestJSONResponse: apigen.BadRequestJSONResponse(*bad)}, nil
	}
	if saved == nil {
		return nil, errors.New("save companion schedule: no row")
	}
	return apigen.PutGuardianCompanionSchedule200JSONResponse(*saved), nil
}

// ListDailyDigests implements GET /guardian/elders/{elderId}/daily-digests.
func (s *Server) ListDailyDigests(ctx context.Context, req apigen.ListDailyDigestsRequestObject) (apigen.ListDailyDigestsResponseObject, error) {
	if ok, err := s.myWard(ctx, req.ElderId); err != nil {
		return nil, err
	} else if !ok {
		return apigen.ListDailyDigests404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}, nil
	}
	limit := int32(defaultDigestCount)
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	rows, err := s.q.ListRecentDigests(ctx, db.ListRecentDigestsParams{ElderID: req.ElderId, MaxCount: limit})
	if err != nil {
		return nil, fmt.Errorf("list digests: %w", err)
	}
	out := apigen.ListDailyDigests200JSONResponse{Digests: make([]apigen.DailyDigest, len(rows))}
	for i, r := range rows {
		out.Digests[i] = toDigest(r)
	}
	return out, nil
}

// defaultDigestCount is how many days the guardian app shows by default.
const defaultDigestCount = 14

func toDigest(d db.DailyDigest) apigen.DailyDigest {
	return apigen.DailyDigest{
		ElderId:         d.ElderID,
		Date:            openapi_types.Date{Time: d.DigestDate.Time},
		SummaryText:     d.SummaryText,
		EmotionFlag:     d.EmotionFlag,
		SessionCount:    d.SessionCount,
		EscalationCount: d.EscalationCount,
		GeneratedAt:     d.GeneratedAt,
		SentAt:          d.SentAt,
	}
}

// ListGuardianOpenEscalations implements GET /guardian/escalations/open.
func (s *Server) ListGuardianOpenEscalations(ctx context.Context, _ apigen.ListGuardianOpenEscalationsRequestObject) (apigen.ListGuardianOpenEscalationsResponseObject, error) {
	g, err := mustGuardian(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListOpenEscalationsForGuardian(ctx, db.ListOpenEscalationsForGuardianParams{
		GuardianID: g.ID, Since: time.Now().Add(-openEscalationWindow),
	})
	if err != nil {
		return nil, fmt.Errorf("list open escalations: %w", err)
	}
	out := apigen.ListGuardianOpenEscalations200JSONResponse{Escalations: make([]apigen.Escalation, len(rows))}
	for i, r := range rows {
		out.Escalations[i] = toEscalation(escalationRow{r.EscalationEvent, r.ElderID, r.ElderName, r.UtteranceText, nil})
	}
	return out, nil
}

// GetGuardianEscalation implements GET /guardian/escalations/{escalationId}.
func (s *Server) GetGuardianEscalation(ctx context.Context, req apigen.GetGuardianEscalationRequestObject) (apigen.GetGuardianEscalationResponseObject, error) {
	r, ok, err := s.wardEscalation(ctx, req.EscalationId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return apigen.GetGuardianEscalation404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}, nil
	}
	return apigen.GetGuardianEscalation200JSONResponse(toEscalation(r)), nil
}

// AckGuardianEscalation implements POST /guardian/escalations/{escalationId}/ack.
func (s *Server) AckGuardianEscalation(ctx context.Context, req apigen.AckGuardianEscalationRequestObject) (apigen.AckGuardianEscalationResponseObject, error) {
	g, err := mustGuardian(ctx)
	if err != nil {
		return nil, err
	}
	r, ok, err := s.wardEscalation(ctx, req.EscalationId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return apigen.AckGuardianEscalation404JSONResponse{NotFoundJSONResponse: apigen.NotFoundJSONResponse(errNotFound)}, nil
	}
	ev, err := s.q.AcknowledgeEscalationByGuardian(ctx, db.AcknowledgeEscalationByGuardianParams{
		ID: req.EscalationId, GuardianID: &g.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("acknowledge escalation: %w", err)
	}
	r.ev = ev
	return apigen.AckGuardianEscalation200JSONResponse(toEscalation(r)), nil
}

// wardEscalation reads an escalation the guardian may see (one raised in a
// companion conversation with an elder of theirs), or reports false.
func (s *Server) wardEscalation(ctx context.Context, id uuid.UUID) (escalationRow, bool, error) {
	g, err := mustGuardian(ctx)
	if err != nil {
		return escalationRow{}, false, err
	}
	d, err := s.q.GetGuardianEscalation(ctx, db.GetGuardianEscalationParams{ID: id, GuardianID: g.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return escalationRow{}, false, nil
	}
	if err != nil {
		return escalationRow{}, false, fmt.Errorf("read escalation: %w", err)
	}
	if d.SessionMode != "COMPANION" {
		// A pickup escalation is the caregiver's to answer; the guardian
		// hears about it in the daily digest.
		return escalationRow{}, false, nil
	}
	return escalationRow{d.EscalationEvent, d.ElderID, d.ElderName, d.UtteranceText, nil}, true, nil
}

// schedule reads an elder's companion settings, or the defaults.
func (s *Server) schedule(ctx context.Context, elderID uuid.UUID) (apigen.CompanionSchedule, error) {
	row, err := s.q.GetCompanionSchedule(ctx, elderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return apigen.CompanionSchedule{
			Enabled: true, CheckInTimes: []string{}, TimeZone: s.deps.Location.String(),
		}, nil
	}
	if err != nil {
		return apigen.CompanionSchedule{}, fmt.Errorf("read companion schedule: %w", err)
	}
	return toSchedule(row), nil
}
