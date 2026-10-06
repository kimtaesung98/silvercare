package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/apigen"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/escalation"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

// otherCaregiver seeds a second center's caregiver and logs them in.
func otherCaregiver(t *testing.T, pool *pgxpool.Pool) client {
	t.Helper()
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	hash, err := auth.HashPassword("pw-87654321")
	if err != nil {
		t.Fatal(err)
	}
	login := "park"
	if _, err := db.New(pool).SetCaregiverLogin(context.Background(), db.SetCaregiverLoginParams{ID: f.CaregiverID, LoginID: &login, PasswordHash: &hash}); err != nil {
		t.Fatal(err)
	}
	c := client{t: t, h: newHandler(pool)}
	r := decode[apigen.CaregiverLoginResponse](t, c.do(http.MethodPost, "/auth/caregiver/login",
		apigen.CaregiverLoginRequest{LoginId: login, Password: "pw-87654321"}), http.StatusOK)
	c.token = r.AccessToken
	return c
}

func TestVisitDestination(t *testing.T) {
	pool, f, c := setup(t, time.Now().Add(time.Hour))
	v := decode[apigen.Visit](t, c.do(http.MethodGet, "/visits/"+f.VisitID.String(), nil), http.StatusOK)
	if v.Destination != nil {
		t.Fatalf("destination before registering the home: %+v", v.Destination)
	}
	if _, err := pool.Exec(context.Background(),
		`UPDATE elder SET home_address = '서울 중구 세종대로 110', home_latitude = 37.5665, home_longitude = 126.978 WHERE id = $1`, f.ElderID); err != nil {
		t.Fatal(err)
	}
	v = decode[apigen.Visit](t, c.do(http.MethodGet, "/visits/"+f.VisitID.String(), nil), http.StatusOK)
	if d := v.Destination; d == nil || d.Latitude != 37.5665 || d.Longitude != 126.978 || *d.Address != "서울 중구 세종대로 110" {
		t.Errorf("destination = %+v", d)
	}
}

func TestEscalationEndpoints(t *testing.T) {
	pool, f, c := setup(t, time.Now().Add(time.Hour))
	sess, utts := testdb.Conversation(t, pool, f, false,
		testdb.Said{Elder: true, Text: "어제 넘어졌어"}, testdb.Said{Elder: true, Text: "다리가 아파"})
	q := db.New(pool)
	var events []db.EscalationEvent
	for _, u := range utts {
		m, _ := escalation.Check(u.Text)
		ev, err := escalation.FromRule(context.Background(), q, sess.ID, u.ID, m)
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, ev)
	}

	open := decode[apigen.EscalationList](t, c.do(http.MethodGet, "/escalations/open", nil), http.StatusOK)
	if len(open.Escalations) != 2 || open.Escalations[0].Id != events[1].ID {
		t.Fatalf("open = %+v (newest first)", open)
	}

	path := "/escalations/" + events[0].ID.String()
	e := decode[apigen.Escalation](t, c.do(http.MethodGet, path, nil), http.StatusOK)
	if e.TriggerType != apigen.FALLMENTION || e.Source != apigen.RULE || *e.UtteranceText != "어제 넘어졌어" ||
		e.Elder.Name != "김순자" || *e.VisitId != f.VisitID || e.AcknowledgedAt != nil {
		t.Errorf("escalation = %+v", e)
	}

	first := decode[apigen.Escalation](t, c.do(http.MethodPost, path+"/ack", nil), http.StatusOK)
	if first.AcknowledgedAt == nil {
		t.Fatal("not acknowledged")
	}
	again := decode[apigen.Escalation](t, c.do(http.MethodPost, path+"/ack", nil), http.StatusOK)
	if !again.AcknowledgedAt.Equal(*first.AcknowledgedAt) {
		t.Errorf("second ack moved the time: %v -> %v", first.AcknowledgedAt, again.AcknowledgedAt)
	}
	open = decode[apigen.EscalationList](t, c.do(http.MethodGet, "/escalations/open", nil), http.StatusOK)
	if len(open.Escalations) != 1 {
		t.Errorf("open after ack = %d, want 1", len(open.Escalations))
	}

	// Another caregiver sees none of it.
	other := otherCaregiver(t, pool)
	decode[apigen.ApiError](t, other.do(http.MethodGet, path, nil), http.StatusNotFound)
	decode[apigen.ApiError](t, other.do(http.MethodPost, path+"/ack", nil), http.StatusNotFound)
	if l := decode[apigen.EscalationList](t, other.do(http.MethodGet, "/escalations/open", nil), http.StatusOK); len(l.Escalations) != 0 {
		t.Errorf("other caregiver sees %d escalations", len(l.Escalations))
	}
	decode[apigen.ApiError](t, c.do(http.MethodGet, "/escalations/"+uuid.NewString(), nil), http.StatusNotFound)
}

func TestBriefingEndpoints(t *testing.T) {
	pool, f, c := setup(t, time.Now().Add(time.Hour))
	sess, _ := testdb.Conversation(t, pool, f, true, testdb.Said{Elder: true, Text: "큰아들이 왔어"})
	path := "/sessions/" + sess.ID.String() + "/briefing"

	if e := decode[apigen.ApiError](t, c.do(http.MethodGet, path, nil), http.StatusNotFound); e.Code != "BRIEFING_NOT_READY" {
		t.Errorf("before the job: %+v", e)
	}

	q := db.New(pool)
	flag := "평소보다 말씀이 많으심"
	if _, err := q.UpsertBriefing(context.Background(), db.UpsertBriefingParams{
		SessionID: sess.ID, SummaryText: "큰아들 이야기를 하심.", TopKeywords: []byte(`["큰아들"]`), EmotionFlag: &flag,
	}); err != nil {
		t.Fatal(err)
	}
	tag := "STABLE"
	if err := q.SetSessionEmotionTag(context.Background(), db.SetSessionEmotionTagParams{ID: sess.ID, Tag: &tag}); err != nil {
		t.Fatal(err)
	}

	b := decode[apigen.Briefing](t, c.do(http.MethodGet, path, nil), http.StatusOK)
	if b.SummaryText != "큰아들 이야기를 하심." || len(b.TopKeywords) != 1 || *b.EmotionFlag != flag ||
		*b.OverallEmotionTag != apigen.STABLE || *b.EscalationCount != 0 || b.ReadByCaregiverAt != nil {
		t.Errorf("briefing = %+v", b)
	}
	read := decode[apigen.Briefing](t, c.do(http.MethodPost, path+"/read", nil), http.StatusOK)
	if read.ReadByCaregiverAt == nil {
		t.Fatal("not marked read")
	}
	again := decode[apigen.Briefing](t, c.do(http.MethodPost, path+"/read", nil), http.StatusOK)
	if !again.ReadByCaregiverAt.Equal(*read.ReadByCaregiverAt) {
		t.Error("second read moved the time")
	}

	other := otherCaregiver(t, pool)
	if e := decode[apigen.ApiError](t, other.do(http.MethodGet, path, nil), http.StatusNotFound); e.Code != "NOT_FOUND" {
		t.Errorf("other caregiver: %+v", e)
	}
	decode[apigen.ApiError](t, other.do(http.MethodPost, path+"/read", nil), http.StatusNotFound)
	decode[apigen.ApiError](t, c.do(http.MethodGet, "/sessions/"+uuid.NewString()+"/briefing", nil), http.StatusNotFound)
}
