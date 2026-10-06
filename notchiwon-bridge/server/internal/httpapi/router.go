// Package httpapi wires the HTTP routes of the server: the REST API
// generated from api/openapi.yaml plus authentication.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/apigen"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/visit"
)

// Deps are what the HTTP layer needs.
type Deps struct {
	Pool   *pgxpool.Pool
	Visits *visit.Service
	Tokens *auth.Tokens
	Logger *slog.Logger
}

// NewRouter returns the root HTTP handler.
func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	s := &Server{deps: d, q: db.New(d.Pool)}
	strict := apigen.NewStrictHandlerWithOptions(s,
		[]apigen.StrictMiddlewareFunc{s.authenticate},
		apigen.StrictHTTPServerOptions{
			RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
				writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
			},
			ResponseErrorHandlerFunc: s.handleError,
		})
	return apigen.HandlerWithOptions(strict, apigen.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
		},
	})
}

// errNotImplemented marks endpoints whose stage has not come yet.
var errNotImplemented = errors.New("not implemented yet")

func (s *Server) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, errNotImplemented) {
		writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "이 기능은 아직 구현되지 않았습니다.")
		return
	}
	s.deps.Logger.ErrorContext(r.Context(), "request failed",
		"method", r.Method, "path", r.URL.Path, "request_id", middleware.GetReqID(r.Context()), "err", err)
	writeError(w, http.StatusInternalServerError, "INTERNAL", "서버 오류가 발생했습니다.")
}

// authenticate checks the credentials an operation's security scheme asks for
// (oapi-codegen marks it in the context) and stores the principal in the context.
func (s *Server) authenticate(next strictnethttp.StrictHTTPHandlerFunc, _ string) strictnethttp.StrictHTTPHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, req any) (any, error) {
		switch {
		case ctx.Value(apigen.CaregiverAuthScopes) != nil:
			c, err := s.caregiverFromRequest(ctx, r)
			if err != nil {
				return s.unauthorized(ctx, w, err)
			}
			ctx = auth.WithCaregiver(ctx, c)
		case ctx.Value(apigen.DeviceAuthScopes) != nil:
			t, err := s.tabletFromRequest(ctx, r)
			if err != nil {
				return s.unauthorized(ctx, w, err)
			}
			ctx = auth.WithTablet(ctx, t)
		}
		return next(ctx, w, r, req)
	}
}

var errUnauthenticated = errors.New("unauthenticated")

func (s *Server) unauthorized(ctx context.Context, w http.ResponseWriter, err error) (any, error) {
	if !errors.Is(err, errUnauthenticated) {
		// A database error while authenticating is a server problem, not bad credentials.
		return nil, err
	}
	writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "인증이 필요합니다.")
	return nil, nil
}

func (s *Server) caregiverFromRequest(ctx context.Context, r *http.Request) (auth.Caregiver, error) {
	tok, ok := auth.BearerToken(r.Header.Get("Authorization"))
	if !ok {
		return auth.Caregiver{}, errUnauthenticated
	}
	id, err := s.deps.Tokens.VerifyCaregiver(tok)
	if err != nil {
		return auth.Caregiver{}, errUnauthenticated
	}
	c, err := s.q.GetCaregiver(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Caregiver{}, errUnauthenticated
	}
	if err != nil {
		return auth.Caregiver{}, err
	}
	return auth.Caregiver{ID: c.ID, CenterID: c.CenterID}, nil
}

func (s *Server) tabletFromRequest(ctx context.Context, r *http.Request) (auth.Tablet, error) {
	tok, ok := auth.BearerToken(r.Header.Get("Authorization"))
	if !ok {
		return auth.Tablet{}, errUnauthenticated
	}
	d, err := s.q.GetActiveDeviceByTokenHash(ctx, auth.HashDeviceToken(tok))
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Tablet{}, errUnauthenticated
	}
	if err != nil {
		return auth.Tablet{}, err
	}
	if d.Kind != "ELDER_TABLET" || d.ElderID == nil {
		return auth.Tablet{}, errUnauthenticated
	}
	if err := s.q.TouchDevice(ctx, d.ID); err != nil {
		s.deps.Logger.WarnContext(ctx, "touch device", "device_id", d.ID, "err", err)
	}
	return auth.Tablet{DeviceID: d.ID, ElderID: *d.ElderID}, nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apigen.ApiError{Code: code, Message: message})
}
