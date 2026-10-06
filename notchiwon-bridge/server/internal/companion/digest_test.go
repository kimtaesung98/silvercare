package companion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/escalation"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/jobs"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/llm"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/notify"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

// fakeDigester records what it was asked to summarize.
type fakeDigester struct {
	calls []llm.DigestInput
	out   llm.Digest
	err   error
}

func (f *fakeDigester) Digest(_ context.Context, in llm.DigestInput) (llm.Digest, error) {
	f.calls = append(f.calls, in)
	if f.err != nil {
		return llm.Digest{}, f.err
	}
	return f.out, nil
}

// fakePush records every push and can fail a given token.
type fakePush struct {
	sent map[string]notify.Message
	err  error
}

func (f *fakePush) Send(_ context.Context, token string, m notify.Message) error {
	if f.err != nil {
		return f.err
	}
	if f.sent == nil {
		f.sent = map[string]notify.Message{}
	}
	f.sent[token] = m
	return nil
}

// digestDay is today in Seoul: the talks the tests record happen now.
var digestDay = time.Now().In(seoul).Format(time.DateOnly)

func digest(t *testing.T, w *DigestWorker, elderID uuid.UUID, date string) error {
	t.Helper()
	return w.Work(context.Background(), &river.Job[jobs.DailyDigestArgs]{
		Args: jobs.DailyDigestArgs{ElderID: elderID, Date: date},
	})
}

// digestWorker wires a worker whose "today" is the digest day.
func digestWorker(q *db.Queries, d llm.Digester, s notify.Sender) *DigestWorker {
	day, _ := time.ParseInLocation(time.DateOnly, digestDay, seoul)
	return &DigestWorker{
		Queries: q, Digester: d, Sender: s, Logger: discard,
		Policy: Policy{Q: q, DefaultTokenLimit: 100000, Location: seoul,
			Now: func() time.Time { return day.Add(21 * time.Hour) }},
	}
}

func TestDigestWritesAndPushes(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	ctx := context.Background()
	f := testdb.Seed(t, pool, time.Now().Add(24*time.Hour))
	testdb.CompanionTalk(t, pool, f, "ELDER_DECLINED",
		testdb.Said{Elder: true, Text: "큰아들이 왔어"},
		testdb.Said{Text: "반가우셨겠어요."},
	)
	phone := testdb.GuardianPhone(t, pool, f.GuardianID, "tok-g1")

	flag := "평소보다 말씀이 많으심"
	d := &fakeDigester{out: llm.Digest{Summary: "큰아들 이야기를 즐겁게 하셨어요.", EmotionFlag: &flag, Model: "claude-haiku-4-5"}}
	push := &fakePush{}
	w := digestWorker(q, d, push)
	if err := digest(t, w, f.ElderID, digestDay); err != nil {
		t.Fatal(err)
	}

	if len(d.calls) != 1 {
		t.Fatalf("calls = %d", len(d.calls))
	}
	if in := d.calls[0]; in.ElderName != "김순자" || in.Date != digestDay || in.SessionCount != 1 || len(in.Lines) != 2 {
		t.Errorf("input = %+v", in)
	}
	row, err := q.GetDailyDigest(ctx, db.GetDailyDigestParams{ElderID: f.ElderID, DigestDate: mustDate(digestDay)})
	if err != nil {
		t.Fatal(err)
	}
	if row.SummaryText != d.out.Summary || row.SessionCount != 1 || row.EscalationCount != 0 ||
		row.Model == nil || *row.Model != "claude-haiku-4-5" || row.SentAt == nil {
		t.Fatalf("row = %+v", row)
	}
	msg, ok := push.sent["tok-g1"]
	if !ok {
		t.Fatalf("pushed to %v", push.sent)
	}
	if msg.Title != "김순자 어르신 하루 소식" || !strings.Contains(msg.Body, "큰아들") ||
		!strings.Contains(msg.Body, flag) || msg.Data["type"] != "digest" || msg.Data["date"] != digestDay ||
		msg.Channel != notify.DigestChannel {
		t.Errorf("message = %+v", msg)
	}
	if phone.FcmToken == nil || *phone.FcmToken != "tok-g1" {
		t.Errorf("phone = %+v", phone)
	}

	// A retry neither rewrites the row nor pushes again.
	push.sent = nil
	if err := digest(t, w, f.ElderID, digestDay); err != nil {
		t.Fatal(err)
	}
	if len(d.calls) != 1 || len(push.sent) != 0 {
		t.Errorf("retry: %d calls, %d pushes", len(d.calls), len(push.sent))
	}
}

func TestDigestQuietDay(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	f := testdb.Seed(t, pool, time.Now().Add(24*time.Hour))
	testdb.GuardianPhone(t, pool, f.GuardianID, "tok-g1")

	d := &fakeDigester{out: llm.Digest{Summary: "묻지 말아야 할 요약"}}
	push := &fakePush{}
	if err := digest(t, digestWorker(q, d, push), f.ElderID, digestDay); err != nil {
		t.Fatal(err)
	}
	if len(d.calls) != 0 {
		t.Errorf("asked Claude about a day with no talk: %+v", d.calls)
	}
	row, err := q.GetDailyDigest(context.Background(), db.GetDailyDigestParams{
		ElderID: f.ElderID, DigestDate: mustDate(digestDay),
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.SummaryText != NoTalkSummary || row.SessionCount != 0 {
		t.Errorf("row = %+v", row)
	}
	if _, ok := push.sent["tok-g1"]; !ok {
		t.Errorf("a quiet day still goes out: %v", push.sent)
	}
}

func TestDigestQuotesElderWithoutClaude(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	f := testdb.Seed(t, pool, time.Now().Add(24*time.Hour))
	testdb.CompanionTalk(t, pool, f, "NO_RESPONSE",
		testdb.Said{Elder: true, Text: "오늘 날씨가 좋네"},
		testdb.Said{Text: "산책 다녀오셨어요?"},
		testdb.Said{Elder: true, Text: "경로당에 갔다 왔지"},
	)

	push := &fakePush{}
	w := digestWorker(q, &fakeDigester{err: llm.ErrUnavailable}, push)
	if err := digest(t, w, f.ElderID, digestDay); err != nil {
		t.Fatalf("no key should not fail the digest: %v", err)
	}
	row, err := q.GetDailyDigest(context.Background(), db.GetDailyDigestParams{
		ElderID: f.ElderID, DigestDate: mustDate(digestDay),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(row.SummaryText, quotedPrefix) ||
		!strings.Contains(row.SummaryText, "경로당에 갔다 왔지") || strings.Contains(row.SummaryText, "산책") {
		t.Errorf("summary = %q", row.SummaryText)
	}
	if row.Model != nil {
		t.Errorf("model = %v, want none", *row.Model)
	}
}

func TestDigestFlagsEscalations(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	ctx := context.Background()
	f := testdb.Seed(t, pool, time.Now().Add(24*time.Hour))
	sess, utts := testdb.CompanionTalk(t, pool, f, "NO_RESPONSE",
		testdb.Said{Elder: true, Text: "어제 넘어졌어"},
		testdb.Said{Elder: true, Text: "또 넘어질까 겁나"},
	)
	for _, u := range utts {
		m, ok := escalation.Check(u.Text)
		if !ok {
			continue
		}
		if _, err := escalation.FromRule(ctx, q, sess.ID, u.ID, m); err != nil {
			t.Fatal(err)
		}
	}

	d := &fakeDigester{err: llm.ErrUnavailable}
	w := digestWorker(q, d, &fakePush{})
	if err := digest(t, w, f.ElderID, digestDay); err != nil {
		t.Fatal(err)
	}
	row, err := q.GetDailyDigest(ctx, db.GetDailyDigestParams{ElderID: f.ElderID, DigestDate: mustDate(digestDay)})
	if err != nil {
		t.Fatal(err)
	}
	if row.EscalationCount == 0 {
		t.Fatalf("escalation count = %d", row.EscalationCount)
	}
	if row.EmotionFlag == nil || !strings.Contains(*row.EmotionFlag, "낙상 언급") ||
		!strings.Contains(*row.EmotionFlag, "바로 연락드렸어요") {
		t.Errorf("flag = %v", row.EmotionFlag)
	}
	// One label per kind, however many events there were.
	if n := strings.Count(*row.EmotionFlag, "낙상 언급"); n != 1 {
		t.Errorf("label repeated %d times: %q", n, *row.EmotionFlag)
	}
}

func TestDigestWithoutGuardianPhone(t *testing.T) {
	pool := testdb.New(t)
	q := db.New(pool)
	f := testdb.Seed(t, pool, time.Now().Add(24*time.Hour))

	w := digestWorker(q, &fakeDigester{out: llm.Digest{Summary: "조용한 하루였어요."}}, &fakePush{})
	if err := digest(t, w, f.ElderID, digestDay); err != nil {
		t.Fatal(err)
	}
	row, err := q.GetDailyDigest(context.Background(), db.GetDailyDigestParams{
		ElderID: f.ElderID, DigestDate: mustDate(digestDay),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Written so the app can show it, but not marked sent.
	if row.SentAt != nil {
		t.Errorf("sent_at = %v with no phone registered", row.SentAt)
	}
}

func TestDigestOfAnUnknownElderIsCancelled(t *testing.T) {
	pool := testdb.New(t)
	w := digestWorker(db.New(pool), &fakeDigester{}, &fakePush{})
	var cancel *river.JobCancelError
	if err := digest(t, w, uuid.New(), digestDay); !errors.As(err, &cancel) {
		t.Errorf("err = %v, want it cancelled", err)
	}
}
