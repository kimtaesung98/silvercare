package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/apigen"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/escalation"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

// guardian logs in the fixture's guardian and returns a client holding the token.
func guardian(t *testing.T, pool *pgxpool.Pool, h http.Handler, f testdb.Fixture, loginID string) (client, apigen.GuardianLoginResponse) {
	t.Helper()
	testdb.GuardianLogin(t, pool, f.GuardianID, loginID, "pw-12345678")
	c := client{t: t, h: h}
	r := decode[apigen.GuardianLoginResponse](t, c.do(http.MethodPost, "/auth/guardian/login",
		apigen.CaregiverLoginRequest{LoginId: loginID, Password: "pw-12345678"}), http.StatusOK)
	c.token = r.AccessToken
	return c, r
}

func TestGuardianLoginAndContext(t *testing.T) {
	pool, f, cg := setup(t, time.Now().Add(time.Hour))
	g, login := guardian(t, pool, cg.h, f, "kim-guardian")

	if login.Guardian.Id != f.GuardianID || len(login.Elders) != 1 || login.Elders[0].Id != f.ElderID {
		t.Fatalf("login = %+v", login)
	}
	me := decode[apigen.GuardianContext](t, g.do(http.MethodGet, "/guardian/me", nil), http.StatusOK)
	if me.Guardian.Id != f.GuardianID || len(me.Elders) != 1 || me.Elders[0].Name != "김순자" {
		t.Errorf("me = %+v", me)
	}

	// A wrong password, and a login id nobody has, look the same.
	decode[apigen.ApiError](t, client{t: t, h: g.h}.do(http.MethodPost, "/auth/guardian/login",
		apigen.CaregiverLoginRequest{LoginId: "kim-guardian", Password: "wrong-password"}), http.StatusUnauthorized)
	decode[apigen.ApiError](t, client{t: t, h: g.h}.do(http.MethodPost, "/auth/guardian/login",
		apigen.CaregiverLoginRequest{LoginId: "nobody", Password: "pw-12345678"}), http.StatusUnauthorized)

	// A guardian token does not open the caregiver's endpoints, nor the
	// other way round.
	decode[apigen.ApiError](t, g.do(http.MethodGet, "/visits/today", nil), http.StatusUnauthorized)
	decode[apigen.ApiError](t, cg.do(http.MethodGet, "/guardian/me", nil), http.StatusUnauthorized)
}

func TestGuardianRegistersPhone(t *testing.T) {
	pool, f, cg := setup(t, time.Now().Add(time.Hour))
	g, _ := guardian(t, pool, cg.h, f, "kim-guardian")

	label := "갤럭시 S25"
	if rec := g.do(http.MethodPut, "/guardian/devices/me/fcm-token",
		apigen.FcmTokenRegistration{FcmToken: "tok-g1", Label: &label}); rec.Code != http.StatusNoContent {
		t.Fatalf("register = %d; %s", rec.Code, rec.Body)
	}
	// Registering the same token again keeps one device.
	if rec := g.do(http.MethodPut, "/guardian/devices/me/fcm-token",
		apigen.FcmTokenRegistration{FcmToken: "tok-g1"}); rec.Code != http.StatusNoContent {
		t.Fatalf("re-register = %d; %s", rec.Code, rec.Body)
	}
	phones, err := db.New(pool).ListGuardianPhones(context.Background(), &f.GuardianID)
	if err != nil {
		t.Fatal(err)
	}
	if len(phones) != 1 || *phones[0].FcmToken != "tok-g1" {
		t.Errorf("phones = %+v", phones)
	}

	decode[apigen.ApiError](t, g.do(http.MethodPut, "/guardian/devices/me/fcm-token",
		apigen.FcmTokenRegistration{FcmToken: "  "}), http.StatusBadRequest)
}

func TestGuardianCompanionSchedule(t *testing.T) {
	pool, f, cg := setup(t, time.Now().Add(time.Hour))
	g, _ := guardian(t, pool, cg.h, f, "kim-guardian")
	path := "/guardian/elders/" + f.ElderID.String() + "/companion-schedule"

	got := decode[apigen.CompanionSchedule](t, g.do(http.MethodGet, path, nil), http.StatusOK)
	if !got.Enabled || len(got.CheckInTimes) != 0 || got.BedtimeStart != nil {
		t.Fatalf("defaults = %+v", got)
	}

	start, end, limit := "21:30", "07:00", int32(40000)
	saved := decode[apigen.CompanionSchedule](t, g.do(http.MethodPut, path, apigen.CompanionSchedule{
		Enabled: true, CheckInTimes: []string{"10:00", "15:30"},
		BedtimeStart: &start, BedtimeEnd: &end, DailyTokenLimit: &limit, TimeZone: "Asia/Seoul",
	}), http.StatusOK)
	if len(saved.CheckInTimes) != 2 || saved.CheckInTimes[0] != "10:00" || *saved.BedtimeStart != "21:30" ||
		*saved.DailyTokenLimit != 40000 || saved.TimeZone != "Asia/Seoul" {
		t.Fatalf("saved = %+v", saved)
	}
	// Read back, and the caregiver sees the same settings.
	again := decode[apigen.CompanionSchedule](t, g.do(http.MethodGet, path, nil), http.StatusOK)
	if len(again.CheckInTimes) != 2 || *again.BedtimeEnd != "07:00" {
		t.Errorf("read back = %+v", again)
	}
	byCaregiver := decode[apigen.CompanionSchedule](t,
		cg.do(http.MethodGet, "/elders/"+f.ElderID.String()+"/companion-schedule", nil), http.StatusOK)
	if len(byCaregiver.CheckInTimes) != 2 {
		t.Errorf("caregiver sees %+v", byCaregiver)
	}

	// A malformed time of day is a bad request, not a saved mess.
	decode[apigen.ApiError](t, g.do(http.MethodPut, path, apigen.CompanionSchedule{
		Enabled: true, CheckInTimes: []string{"25:00"}, TimeZone: "Asia/Seoul",
	}), http.StatusBadRequest)
	// So is a time zone nobody has.
	decode[apigen.ApiError](t, g.do(http.MethodPut, path, apigen.CompanionSchedule{
		Enabled: true, CheckInTimes: []string{}, TimeZone: "Mars/Olympus",
	}), http.StatusBadRequest)

	// Another guardian's elder is not theirs to read or change.
	other := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	og, _ := guardian(t, pool, cg.h, other, "park-guardian")
	decode[apigen.ApiError](t, og.do(http.MethodGet, path, nil), http.StatusNotFound)
	decode[apigen.ApiError](t, og.do(http.MethodPut, path, apigen.CompanionSchedule{
		Enabled: false, CheckInTimes: []string{}, TimeZone: "Asia/Seoul",
	}), http.StatusNotFound)
	decode[apigen.ApiError](t, g.do(http.MethodGet,
		"/guardian/elders/"+uuid.NewString()+"/companion-schedule", nil), http.StatusNotFound)
}

func TestGuardianDailyDigests(t *testing.T) {
	pool, f, cg := setup(t, time.Now().Add(time.Hour))
	g, _ := guardian(t, pool, cg.h, f, "kim-guardian")
	q := db.New(pool)
	ctx := context.Background()

	for i, day := range []string{"2026-10-03", "2026-10-04", "2026-10-05"} {
		at, err := time.Parse("2006-01-02", day)
		if err != nil {
			t.Fatal(err)
		}
		flag := "STABLE"
		if _, err := q.InsertDailyDigest(ctx, db.InsertDailyDigestParams{
			ElderID: f.ElderID, DigestDate: pgtype.Date{Time: at, Valid: true},
			SummaryText: day + " 이야기", EmotionFlag: &flag,
			SessionCount: int32(i + 1), EscalationCount: 0,
		}); err != nil {
			t.Fatal(err)
		}
	}
	path := "/guardian/elders/" + f.ElderID.String() + "/daily-digests"
	list := decode[apigen.DailyDigestList](t, g.do(http.MethodGet, path, nil), http.StatusOK)
	if len(list.Digests) != 3 {
		t.Fatalf("digests = %+v", list)
	}
	if got := list.Digests[0].Date.Format("2006-01-02"); got != "2026-10-05" {
		t.Errorf("newest first: %s", got)
	}
	if d := list.Digests[0]; d.SummaryText != "2026-10-05 이야기" || d.SessionCount != 3 || d.SentAt != nil {
		t.Errorf("newest = %+v", d)
	}

	limited := decode[apigen.DailyDigestList](t, g.do(http.MethodGet, path+"?limit=1", nil), http.StatusOK)
	if len(limited.Digests) != 1 || limited.Digests[0].SummaryText != "2026-10-05 이야기" {
		t.Errorf("limit=1 gave %+v", limited)
	}

	other := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	og, _ := guardian(t, pool, cg.h, other, "park-guardian")
	decode[apigen.ApiError](t, og.do(http.MethodGet, path, nil), http.StatusNotFound)
}

func TestGuardianEscalations(t *testing.T) {
	pool, f, cg := setup(t, time.Now().Add(time.Hour))
	g, _ := guardian(t, pool, cg.h, f, "kim-guardian")
	q := db.New(pool)
	ctx := context.Background()

	// One escalation from a companion talk (the guardian's), one from the
	// pickup bridge (the caregiver's).
	talk, talkUtts := testdb.CompanionTalk(t, pool, f, "NO_RESPONSE", testdb.Said{Elder: true, Text: "가슴이 아파"})
	m, _ := escalation.Check(talkUtts[0].Text)
	mine, err := escalation.FromRule(ctx, q, talk.ID, talkUtts[0].ID, m)
	if err != nil {
		t.Fatal(err)
	}
	pickup, pickupUtts := testdb.Conversation(t, pool, f, false, testdb.Said{Elder: true, Text: "어제 넘어졌어"})
	pm, _ := escalation.Check(pickupUtts[0].Text)
	theirs, err := escalation.FromRule(ctx, q, pickup.ID, pickupUtts[0].ID, pm)
	if err != nil {
		t.Fatal(err)
	}

	open := decode[apigen.EscalationList](t, g.do(http.MethodGet, "/guardian/escalations/open", nil), http.StatusOK)
	if len(open.Escalations) != 1 || open.Escalations[0].Id != mine.ID {
		t.Fatalf("open = %+v (companion only)", open)
	}
	if e := open.Escalations[0]; e.VisitId != nil || e.Elder.Name != "김순자" {
		t.Errorf("escalation = %+v", e)
	}

	path := "/guardian/escalations/" + mine.ID.String()
	decode[apigen.Escalation](t, g.do(http.MethodGet, path, nil), http.StatusOK)
	// The pickup one is not the guardian's to see or acknowledge.
	decode[apigen.ApiError](t, g.do(http.MethodGet, "/guardian/escalations/"+theirs.ID.String(), nil), http.StatusNotFound)
	decode[apigen.ApiError](t, g.do(http.MethodPost, "/guardian/escalations/"+theirs.ID.String()+"/ack", nil), http.StatusNotFound)

	acked := decode[apigen.Escalation](t, g.do(http.MethodPost, path+"/ack", nil), http.StatusOK)
	if acked.AcknowledgedAt == nil {
		t.Fatal("not acknowledged")
	}
	again := decode[apigen.Escalation](t, g.do(http.MethodPost, path+"/ack", nil), http.StatusOK)
	if !again.AcknowledgedAt.Equal(*acked.AcknowledgedAt) {
		t.Errorf("second ack moved the time: %v -> %v", acked.AcknowledgedAt, again.AcknowledgedAt)
	}
	row, err := q.GetEscalationDetail(ctx, mine.ID)
	if err != nil {
		t.Fatal(err)
	}
	if row.EscalationEvent.AcknowledgedByGuardianID == nil || *row.EscalationEvent.AcknowledgedByGuardianID != f.GuardianID {
		t.Errorf("acknowledged_by_guardian_id = %v", row.EscalationEvent.AcknowledgedByGuardianID)
	}
	if l := decode[apigen.EscalationList](t, g.do(http.MethodGet, "/guardian/escalations/open", nil), http.StatusOK); len(l.Escalations) != 0 {
		t.Errorf("open after ack = %+v", l)
	}

	other := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	og, _ := guardian(t, pool, cg.h, other, "park-guardian")
	decode[apigen.ApiError](t, og.do(http.MethodGet, path, nil), http.StatusNotFound)
}
