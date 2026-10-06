package notify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/escalation"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/jobs"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestFCMSend(t *testing.T) {
	var (
		path string
		body map[string]any
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		switch {
		case strings.Contains(string(b), "dead-token"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"status":"NOT_FOUND","details":[{"errorCode":"UNREGISTERED"}]}}`))
		case strings.Contains(string(b), "busy-token"):
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			_, _ = w.Write([]byte(`{"name":"projects/p/messages/1"}`))
		}
	}))
	defer srv.Close()
	f := newFCMForTest("silvercare-dev", srv.URL, srv.Client())

	msg := Message{Title: "위급 알림", Body: "통증 호소", Data: map[string]string{"type": "escalation"}}
	if err := f.Send(context.Background(), "good-token", msg); err != nil {
		t.Fatal(err)
	}
	if path != "/v1/projects/silvercare-dev/messages:send" {
		t.Errorf("path = %s", path)
	}
	m := body["message"].(map[string]any)
	android := m["android"].(map[string]any)
	if m["token"] != "good-token" || android["priority"] != "HIGH" ||
		android["notification"].(map[string]any)["channel_id"] != AndroidChannel ||
		m["data"].(map[string]any)["type"] != "escalation" {
		t.Errorf("message = %v", m)
	}

	if err := f.Send(context.Background(), "dead-token", msg); !errors.Is(err, ErrUnregistered) {
		t.Errorf("dead token: %v, want ErrUnregistered", err)
	}
	if err := f.Send(context.Background(), "busy-token", msg); err == nil || errors.Is(err, ErrUnregistered) {
		t.Errorf("503: %v, want a retryable error", err)
	}
}

// fakeSender records sends and fails for chosen tokens.
type fakeSender struct {
	mu    sync.Mutex
	sent  []string
	msgs  []Message
	fails map[string]error
}

func (f *fakeSender) Send(_ context.Context, token string, m Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fails[token]; err != nil {
		return err
	}
	f.sent = append(f.sent, token)
	f.msgs = append(f.msgs, m)
	return nil
}

// raise records a rule escalation on "다리가 아파" in the fixture's pickup session.
func raise(t *testing.T, pool *pgxpool.Pool, f testdb.Fixture) db.EscalationEvent {
	t.Helper()
	sess, utts := testdb.Conversation(t, pool, f, false, testdb.Said{Elder: true, Text: "다리가 아파"})
	m, ok := escalation.Check(utts[0].Text)
	if !ok {
		t.Fatal("rule did not match")
	}
	ev, err := escalation.FromRule(context.Background(), db.New(pool), sess.ID, utts[0].ID, m)
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func work(t *testing.T, w *EscalationWorker, ev db.EscalationEvent) error {
	t.Helper()
	return w.Work(context.Background(), &river.Job[jobs.EscalationAlertArgs]{Args: jobs.EscalationAlertArgs{EscalationID: ev.ID}})
}

func TestEscalationWorkerPushesCaregiver(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	live := testdb.Phone(t, pool, f.CaregiverID, "live-token")
	dead := testdb.Phone(t, pool, f.CaregiverID, "dead-token")
	ev := raise(t, pool, f)
	q := db.New(pool)

	s := &fakeSender{fails: map[string]error{"dead-token": ErrUnregistered}}
	if err := work(t, &EscalationWorker{Queries: q, Sender: s, Logger: discard}, ev); err != nil {
		t.Fatal(err)
	}
	if len(s.sent) != 1 || s.sent[0] != "live-token" {
		t.Fatalf("sent = %v", s.sent)
	}
	m := s.msgs[0]
	if m.Title != "위급 알림: 김순자 어르신" || m.Body != "통증 호소: “다리가 아파”" ||
		m.Data["escalationId"] != ev.ID.String() || m.Data["visitId"] != f.VisitID.String() {
		t.Errorf("message = %+v", m)
	}

	got, err := q.GetEscalationEvent(context.Background(), ev.ID)
	if err != nil {
		t.Fatal(err)
	}
	var targets []Target
	if err := json.Unmarshal(got.NotifiedTargets, &targets); err != nil || len(targets) != 1 || targets[0].DeviceID != live.ID {
		t.Errorf("notified_targets = %s", got.NotifiedTargets)
	}
	phones, err := q.ListCaregiverPhones(context.Background(), &f.CaregiverID)
	if err != nil || len(phones) != 1 || phones[0].ID == dead.ID {
		t.Errorf("phones after = %+v, %v (dead token should be revoked)", phones, err)
	}
}

func TestEscalationWorkerRetriesWhenNobodyGotIt(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	testdb.Phone(t, pool, f.CaregiverID, "busy-token")
	ev := raise(t, pool, f)

	s := &fakeSender{fails: map[string]error{"busy-token": errors.New("503")}}
	if err := work(t, &EscalationWorker{Queries: db.New(pool), Sender: s, Logger: discard}, ev); err == nil {
		t.Fatal("want an error so River retries")
	}
}

func TestEscalationWorkerSkips(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	q := db.New(pool)

	// No phone registered: nothing to do, no retry.
	ev := raise(t, pool, f)
	s := &fakeSender{}
	if err := work(t, &EscalationWorker{Queries: q, Sender: s, Logger: discard}, ev); err != nil || len(s.sent) != 0 {
		t.Fatalf("no phone: %v, sent %v", err, s.sent)
	}

	// FCM not configured: no retry either.
	testdb.Phone(t, pool, f.CaregiverID, "live-token")
	if err := work(t, &EscalationWorker{Queries: q, Sender: Unavailable{}, Logger: discard}, ev); err != nil {
		t.Fatalf("unavailable: %v", err)
	}

	// Acknowledged before the (retried) job ran: no push.
	if _, err := q.AcknowledgeEscalation(context.Background(), db.AcknowledgeEscalationParams{ID: ev.ID, CaregiverID: &f.CaregiverID}); err != nil {
		t.Fatal(err)
	}
	if err := work(t, &EscalationWorker{Queries: q, Sender: s, Logger: discard}, ev); err != nil || len(s.sent) != 0 {
		t.Fatalf("acknowledged: %v, sent %v", err, s.sent)
	}
}

func TestEscalationMessageUsesReasonForLLM(t *testing.T) {
	reason := "집을 못 찾겠다고 하심"
	m := EscalationMessage(db.GetEscalationDetailRow{
		EscalationEvent: db.EscalationEvent{TriggerType: "OTHER_ANOMALY", Source: "LLM", Reason: &reason},
		ElderName:       "김순자",
	})
	if m.Body != "이상 징후: 집을 못 찾겠다고 하심" || m.Data["visitId"] != "" {
		t.Errorf("message = %+v", m)
	}
}
