package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/apigen"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/eta"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/opener"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/speech"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/visit"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

func newHandler(pool *pgxpool.Pool) http.Handler {
	return newHandlerWithTTS(pool, speech.Unavailable{})
}

func newHandlerWithTTS(pool *pgxpool.Pool, tts speech.Synthesizer) http.Handler {
	return NewRouter(Deps{
		Pool:    pool,
		Visits:  visit.NewService(pool, eta.Schedule{}, nil, visit.Config{TriggerEtaMinutes: 15, Location: time.UTC}, discard),
		Tokens:  auth.NewTokens(strings.Repeat("k", 32), time.Hour),
		Openers: opener.NewLibrary(db.New(pool)),
		TTS:     tts,
		Logger:  discard,
	})
}

// countingTTS returns the text as "audio" and counts calls.
type countingTTS struct{ calls int }

func (c *countingTTS) Synthesize(_ context.Context, text, voice string) ([]byte, error) {
	c.calls++
	return []byte("mp3:" + voice + ":" + text), nil
}

type client struct {
	t     *testing.T
	h     http.Handler
	token string
}

func (c client) do(method, path string, body any) *httptest.ResponseRecorder {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) T {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, wantStatus, rec.Body)
	}
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %s: %v", rec.Body, err)
	}
	return v
}

func TestHealthz(t *testing.T) {
	rec := client{t: t, h: newHandler(nil)}.do(http.MethodGet, "/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got, want := rec.Body.String(), "{\"status\":\"ok\"}\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestCaregiverEndpointsNeedToken(t *testing.T) {
	h := newHandler(nil)
	for _, tok := range []string{"", "garbage"} {
		rec := client{t: t, h: h, token: tok}.do(http.MethodGet, "/visits/today", nil)
		e := decode[apigen.ApiError](t, rec, http.StatusUnauthorized)
		if e.Code != "UNAUTHENTICATED" {
			t.Errorf("code %q", e.Code)
		}
	}
}

// setup seeds a caregiver with a login and returns a logged-in client.
func setup(t *testing.T, visitAt time.Time) (*pgxpool.Pool, testdb.Fixture, client) {
	t.Helper()
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, visitAt)
	hash, err := auth.HashPassword("pw-12345678")
	if err != nil {
		t.Fatal(err)
	}
	login := "lee"
	if _, err := db.New(pool).SetCaregiverLogin(context.Background(), db.SetCaregiverLoginParams{
		ID: f.CaregiverID, LoginID: &login, PasswordHash: &hash,
	}); err != nil {
		t.Fatal(err)
	}
	c := client{t: t, h: newHandler(pool)}

	rec := c.do(http.MethodPost, "/auth/caregiver/login", apigen.CaregiverLoginRequest{LoginId: "lee", Password: "wrong"})
	decode[apigen.ApiError](t, rec, http.StatusUnauthorized)
	rec = c.do(http.MethodPost, "/auth/caregiver/login", apigen.CaregiverLoginRequest{LoginId: "nobody", Password: "pw-12345678"})
	decode[apigen.ApiError](t, rec, http.StatusUnauthorized)

	rec = c.do(http.MethodPost, "/auth/caregiver/login", apigen.CaregiverLoginRequest{LoginId: "lee", Password: "pw-12345678"})
	login200 := decode[apigen.CaregiverLoginResponse](t, rec, http.StatusOK)
	if login200.Caregiver.Id != f.CaregiverID {
		t.Fatalf("logged in as %v", login200.Caregiver.Id)
	}
	c.token = login200.AccessToken
	return pool, f, c
}

func TestVisitFlow(t *testing.T) {
	_, f, c := setup(t, time.Now().Add(10*time.Minute))

	list := decode[apigen.VisitList](t, c.do(http.MethodGet, "/visits/today", nil), http.StatusOK)
	if len(list.Visits) != 1 || list.Visits[0].Id != f.VisitID || list.Visits[0].Status != apigen.SCHEDULED {
		t.Fatalf("today: %+v", list)
	}

	loc := apigen.LocationUpdate{Latitude: 37.56, Longitude: 126.97, RecordedAt: time.Now()}
	path := "/visits/" + f.VisitID.String()
	res := decode[apigen.LocationUpdateResult](t, c.do(http.MethodPost, path+"/location", loc), http.StatusOK)
	if !res.SessionStarted || res.Status != apigen.SESSIONACTIVE || res.SessionId == nil || res.EtaMinutes > 15 {
		t.Fatalf("location: %+v", res)
	}

	v := decode[apigen.Visit](t, c.do(http.MethodGet, path, nil), http.StatusOK)
	if v.SessionId == nil || *v.SessionId != *res.SessionId || v.EtaCurrent == nil {
		t.Fatalf("visit: %+v", v)
	}

	v = decode[apigen.Visit](t, c.do(http.MethodPost, path+"/arrive", nil), http.StatusOK)
	if v.Status != apigen.COMPLETED || v.ActualArrivalTime == nil {
		t.Fatalf("arrive: %+v", v)
	}

	e := decode[apigen.ApiError](t, c.do(http.MethodPost, path+"/location", loc), http.StatusConflict)
	if e.Code != "VISIT_CLOSED" {
		t.Errorf("after arrival: %+v", e)
	}
	decode[apigen.ApiError](t, c.do(http.MethodPost, path+"/arrive", nil), http.StatusConflict)
	decode[apigen.ApiError](t, c.do(http.MethodGet, "/visits/"+uuid.NewString(), nil), http.StatusNotFound)
}

func TestLocationValidation(t *testing.T) {
	_, f, c := setup(t, time.Now().Add(time.Hour))
	path := "/visits/" + f.VisitID.String() + "/location"
	for name, body := range map[string]any{
		"latitude":    apigen.LocationUpdate{Latitude: 91, Longitude: 126, RecordedAt: time.Now()},
		"longitude":   apigen.LocationUpdate{Latitude: 37, Longitude: -181, RecordedAt: time.Now()},
		"no time":     map[string]any{"latitude": 37, "longitude": 126},
		"not json":    "nope",
		"bad visitId": nil,
	} {
		t.Run(name, func(t *testing.T) {
			p := path
			if name == "bad visitId" {
				p = "/visits/not-a-uuid/location"
				body = apigen.LocationUpdate{Latitude: 37, Longitude: 126, RecordedAt: time.Now()}
			}
			e := decode[apigen.ApiError](t, c.do(http.MethodPost, p, body), http.StatusBadRequest)
			if e.Code != "INVALID_ARGUMENT" {
				t.Errorf("code %q", e.Code)
			}
		})
	}
}

func TestFcmToken(t *testing.T) {
	pool, f, c := setup(t, time.Now().Add(time.Hour))
	rec := c.do(http.MethodPut, "/devices/me/fcm-token", apigen.FcmTokenRegistration{FcmToken: "fcm-1"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d %s", rec.Code, rec.Body)
	}
	// The same token again is an update, not a second device.
	c.do(http.MethodPut, "/devices/me/fcm-token", apigen.FcmTokenRegistration{FcmToken: "fcm-1"})
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM device WHERE caregiver_id = $1`, f.CaregiverID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("devices = %d, want 1", n)
	}
	decode[apigen.ApiError](t, c.do(http.MethodPut, "/devices/me/fcm-token", apigen.FcmTokenRegistration{FcmToken: " "}), http.StatusBadRequest)
}

func TestTabletContext(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	ctx := context.Background()
	token := auth.NewDeviceToken()
	if _, err := db.New(pool).CreateElderTablet(ctx, db.CreateElderTabletParams{
		ElderID: &f.ElderID, TokenHash: auth.HashDeviceToken(token),
	}); err != nil {
		t.Fatal(err)
	}
	h := newHandler(pool)

	decode[apigen.ApiError](t, client{t: t, h: h, token: "wrong"}.do(http.MethodGet, "/tablet/me", nil), http.StatusUnauthorized)

	tablet := client{t: t, h: h, token: token}
	got := decode[apigen.TabletContext](t, tablet.do(http.MethodGet, "/tablet/me", nil), http.StatusOK)
	if got.Elder.Id != f.ElderID || got.ActiveSession != nil || got.CompanionEnabled {
		t.Fatalf("%+v", got)
	}

	if _, err := db.New(pool).CreateCompanionSession(ctx, db.CreateCompanionSessionParams{ElderID: f.ElderID, StartedBy: "ELDER"}); err != nil {
		t.Fatal(err)
	}
	got = decode[apigen.TabletContext](t, tablet.do(http.MethodGet, "/tablet/me", nil), http.StatusOK)
	if got.ActiveSession == nil || got.ActiveSession.Mode != apigen.COMPANION {
		t.Fatalf("%+v", got)
	}

	// A tablet token is not a caregiver token, and the other way round.
	decode[apigen.ApiError](t, tablet.do(http.MethodGet, "/visits/today", nil), http.StatusUnauthorized)

	if _, err := pool.Exec(ctx, `UPDATE device SET revoked_at = now()`); err != nil {
		t.Fatal(err)
	}
	decode[apigen.ApiError](t, tablet.do(http.MethodGet, "/tablet/me", nil), http.StatusUnauthorized)
}

func TestOpenerClips(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	ctx := context.Background()
	token := auth.NewDeviceToken()
	if _, err := db.New(pool).CreateElderTablet(ctx, db.CreateElderTabletParams{
		ElderID: &f.ElderID, TokenHash: auth.HashDeviceToken(token),
	}); err != nil {
		t.Fatal(err)
	}
	tablet := client{t: t, h: newHandler(pool), token: token}

	got := decode[apigen.OpenerClipManifest](t, tablet.do(http.MethodGet, "/tablet/opener-clips", nil), http.StatusOK)
	if got.Voice != opener.DefaultVoice || got.Version == "" || len(got.Clips) == 0 {
		t.Fatalf("%+v", got)
	}
	categories := map[apigen.OpenerCategory]bool{}
	for _, c := range got.Clips {
		categories[c.Category] = true
		if c.AudioUrl != "/tablet/opener-clips/"+c.Id.String()+"/audio" || c.DurationMs <= 0 || c.Text == "" {
			t.Errorf("clip %+v", c)
		}
	}
	if len(categories) != 7 {
		t.Errorf("categories = %v", categories)
	}

	// An elder whose voice has no clips yet gets the default ones.
	if _, err := pool.Exec(ctx, `UPDATE elder SET preferred_tts_voice = 'nara' WHERE id = $1`, f.ElderID); err != nil {
		t.Fatal(err)
	}
	again := decode[apigen.OpenerClipManifest](t, tablet.do(http.MethodGet, "/tablet/opener-clips", nil), http.StatusOK)
	if again.Voice != opener.DefaultVoice || again.Version != got.Version {
		t.Errorf("fallback manifest = %s %s", again.Voice, again.Version)
	}

	// Retiring a clip changes the version.
	if _, err := pool.Exec(ctx, `UPDATE opener_clip SET active = false WHERE id = $1`, got.Clips[0].Id); err != nil {
		t.Fatal(err)
	}
	changed := decode[apigen.OpenerClipManifest](t, tablet.do(http.MethodGet, "/tablet/opener-clips", nil), http.StatusOK)
	if changed.Version == got.Version || len(changed.Clips) != len(got.Clips)-1 {
		t.Errorf("after retiring: %s, %d clips", changed.Version, len(changed.Clips))
	}
}

func TestOpenerClipAudio(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	ctx := context.Background()
	token := auth.NewDeviceToken()
	if _, err := db.New(pool).CreateElderTablet(ctx, db.CreateElderTabletParams{
		ElderID: &f.ElderID, TokenHash: auth.HashDeviceToken(token),
	}); err != nil {
		t.Fatal(err)
	}
	tts := &countingTTS{}
	tablet := client{t: t, h: newHandlerWithTTS(pool, tts), token: token}
	m := decode[apigen.OpenerClipManifest](t, tablet.do(http.MethodGet, "/tablet/opener-clips", nil), http.StatusOK)
	clip := m.Clips[0]
	path := "/tablet/opener-clips/" + clip.Id.String() + "/audio"

	for range 2 {
		rec := tablet.do(http.MethodGet, path, nil)
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "audio/mpeg" || rec.Body.String() != "mp3:default:"+clip.Text {
			t.Fatalf("audio: %d %s %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body)
		}
	}
	if tts.calls != 1 {
		t.Errorf("synthesized %d times, want once", tts.calls)
	}
	// Synthesis changes the manifest version so tablets fetch the new audio.
	after := decode[apigen.OpenerClipManifest](t, tablet.do(http.MethodGet, "/tablet/opener-clips", nil), http.StatusOK)
	if after.Version == m.Version {
		t.Error("version unchanged after synthesis")
	}

	decode[apigen.ApiError](t, tablet.do(http.MethodGet, "/tablet/opener-clips/"+uuid.NewString()+"/audio", nil), http.StatusNotFound)
	other := client{t: t, h: newHandler(pool), token: token}
	if e := decode[apigen.ApiError](t, other.do(http.MethodGet, "/tablet/opener-clips/"+m.Clips[1].Id.String()+"/audio", nil), http.StatusNotFound); e.Code != "AUDIO_NOT_READY" {
		t.Errorf("without tts: %+v", e)
	}
	decode[apigen.ApiError](t, client{t: t, h: newHandler(pool)}.do(http.MethodGet, path, nil), http.StatusUnauthorized)
}

func TestLaterStageEndpointsAnswer501(t *testing.T) {
	_, f, c := setup(t, time.Now().Add(time.Hour))
	e := decode[apigen.ApiError](t, c.do(http.MethodGet, "/elders/"+f.ElderID.String()+"/companion-schedule", nil), http.StatusNotImplemented)
	if e.Code != "NOT_IMPLEMENTED" {
		t.Errorf("code %q", e.Code)
	}
}
