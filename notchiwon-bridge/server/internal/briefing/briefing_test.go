package briefing

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/escalation"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/jobs"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/llm"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

// fakeSummarizer returns a fixed briefing and remembers its input.
type fakeSummarizer struct {
	out   llm.Briefing
	err   error
	calls int
	in    llm.BriefingInput
}

func (f *fakeSummarizer) Brief(_ context.Context, in llm.BriefingInput) (llm.Briefing, error) {
	f.calls++
	f.in = in
	return f.out, f.err
}

func run(t *testing.T, pool *pgxpool.Pool, s llm.Summarizer, sess db.ConversationSession) error {
	t.Helper()
	w := &Worker{Pool: pool, Summarizer: s, Logger: discard}
	return w.Work(context.Background(), &river.Job[jobs.BriefingArgs]{Args: jobs.BriefingArgs{SessionID: sess.ID}})
}

var chat = []testdb.Said{
	{Elder: false, Text: "안녕하세요, 김순자님."},
	{Elder: false, Text: "오늘 기분은 어떠세요?"},
	{Elder: true, Text: "큰아들이 어제 왔어"},
	{Elder: false, Text: "아이고, 반가우셨겠어요."},
	{Elder: true, Text: "화투도 쳤지"},
}

func TestWorkerWritesBriefingAndKeywords(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	sess, _ := testdb.Conversation(t, pool, f, true, chat...)
	flag := "평소보다 말씀이 많으심"
	s := &fakeSummarizer{out: llm.Briefing{
		Summary: "큰아들 이야기를 즐겁게 하심.",
		Keywords: []llm.Keyword{
			{Keyword: "큰아들", Category: "FAMILY", EmotionTone: "POSITIVE", Mentions: 2},
			{Keyword: "화투", Category: "HOBBY", EmotionTone: "POSITIVE", Mentions: 1},
		},
		EmotionTag: "STABLE", EmotionFlag: &flag, Model: "claude-haiku-4-5",
	}}
	if err := run(t, pool, s, sess); err != nil {
		t.Fatal(err)
	}

	// The AI's two sentences of one turn reach Claude as one line.
	if len(s.in.Lines) != 4 || s.in.Lines[0].Text != "안녕하세요, 김순자님. 오늘 기분은 어떠세요?" || !s.in.Lines[1].Elder || s.in.ElderName != "김순자" {
		t.Errorf("summarizer input = %+v", s.in)
	}

	q := db.New(pool)
	ctx := context.Background()
	d, err := q.GetBriefingDetail(ctx, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	var top []string
	_ = json.Unmarshal(d.BriefingReport.TopKeywords, &top)
	if d.BriefingReport.SummaryText != "큰아들 이야기를 즐겁게 하심." || strings.Join(top, ",") != "큰아들,화투" ||
		*d.BriefingReport.EmotionFlag != flag || *d.BriefingReport.Model != "claude-haiku-4-5" ||
		d.OverallEmotionTag == nil || *d.OverallEmotionTag != "STABLE" {
		t.Errorf("briefing = %+v, tag %v", d.BriefingReport, d.OverallEmotionTag)
	}

	kws, err := q.ListTopKeywords(ctx, db.ListTopKeywordsParams{ElderID: f.ElderID, MaxCount: 10})
	if err != nil || len(kws) != 2 || kws[0].Keyword != "큰아들" || kws[0].Score != 2 {
		t.Fatalf("keywords = %+v, %v", kws, err)
	}

	// A retried job leaves the briefing and the scores alone.
	if err := run(t, pool, s, sess); err != nil || s.calls != 1 {
		t.Fatalf("rerun: %v, calls %d", err, s.calls)
	}
	kws, _ = q.ListTopKeywords(ctx, db.ListTopKeywordsParams{ElderID: f.ElderID, MaxCount: 10})
	if kws[0].Score != 2 {
		t.Errorf("score after rerun = %v, want 2 (not bumped twice)", kws[0].Score)
	}
}

func TestWorkerDecaysKeywordsAcrossSessions(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	q := db.New(pool)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	s := &fakeSummarizer{out: llm.Briefing{Summary: "요약", EmotionTag: "STABLE",
		Keywords: []llm.Keyword{{Keyword: "큰아들", Category: "FAMILY", EmotionTone: "POSITIVE", Mentions: 2}}}}
	sess, _ := testdb.Conversation(t, pool, f, true, chat...)
	if err := run(t, pool, s, sess); err != nil {
		t.Fatal(err)
	}

	// The next day's visit mentions the son once more: 2 * 0.9 + 1.
	var visit2 = testdb.Seed(t, pool, time.Now().Add(25*time.Hour))
	if _, err := pool.Exec(ctx, `UPDATE visit SET elder_id = $1 WHERE id = $2`, f.ElderID, visit2.VisitID); err != nil {
		t.Fatal(err)
	}
	visit2.ElderID = f.ElderID
	sess2, _ := testdb.Conversation(t, pool, visit2, true, chat...)
	s.out.Keywords[0].Mentions = 1
	if err := run(t, pool, s, sess2); err != nil {
		t.Fatal(err)
	}
	kws, _ := q.ListTopKeywords(ctx, db.ListTopKeywordsParams{ElderID: f.ElderID, MaxCount: 10})
	if len(kws) != 1 || kws[0].Score < 2.79 || kws[0].Score > 2.81 || kws[0].MentionCountTotal != 3 {
		t.Errorf("keyword = %+v, want score 2.8 and 3 mentions", kws)
	}
}

func TestWorkerWithoutClaudeQuotesTheElder(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	sess, utts := testdb.Conversation(t, pool, f, false, append(chat, testdb.Said{Elder: true, Text: "다리가 아파"})...)
	m, _ := escalation.Check("다리가 아파")
	if _, err := escalation.FromRule(context.Background(), db.New(pool), sess.ID, utts[len(utts)-1].ID, m); err != nil {
		t.Fatal(err)
	}
	reason := "CAREGIVER_ARRIVED"
	if _, err := db.New(pool).EndSession(context.Background(), db.EndSessionParams{ID: sess.ID, EndedReason: &reason}); err != nil {
		t.Fatal(err)
	}

	if err := run(t, pool, llm.Unavailable{}, sess); err != nil {
		t.Fatal(err)
	}
	d, err := db.New(pool).GetBriefingDetail(context.Background(), sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := quotedPrefix + "“큰아들이 어제 왔어”, “화투도 쳤지”, “다리가 아파”"
	if d.BriefingReport.SummaryText != want {
		t.Errorf("summary = %q\nwant      %q", d.BriefingReport.SummaryText, want)
	}
	if *d.OverallEmotionTag != "UNUSUAL" || *d.BriefingReport.EmotionFlag != "위급 감지: 통증 호소" || d.EscalationCount != 1 {
		t.Errorf("tag %v, flag %v, escalations %d", *d.OverallEmotionTag, *d.BriefingReport.EmotionFlag, d.EscalationCount)
	}
}

func TestWorkerSilentSession(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	sess, _ := testdb.Conversation(t, pool, f, true, testdb.Said{Text: "안녕하세요, 김순자님."})
	s := &fakeSummarizer{}
	if err := run(t, pool, s, sess); err != nil {
		t.Fatal(err)
	}
	d, err := db.New(pool).GetBriefingDetail(context.Background(), sess.ID)
	if err != nil || d.BriefingReport.SummaryText != SilentSummary || s.calls != 0 {
		t.Errorf("briefing = %+v, %v, calls %d (no Claude call for a silent session)", d.BriefingReport, err, s.calls)
	}
}

func TestWorkerRetries(t *testing.T) {
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))

	open, _ := testdb.Conversation(t, pool, f, false, chat...)
	if err := run(t, pool, &fakeSummarizer{}, open); err == nil {
		t.Error("session still open: want an error so the job retries")
	}
	reason := "CAREGIVER_ARRIVED"
	ended, err := db.New(pool).EndSession(context.Background(), db.EndSessionParams{ID: open.ID, EndedReason: &reason})
	if err != nil {
		t.Fatal(err)
	}
	if err := run(t, pool, &fakeSummarizer{err: errors.New("overloaded")}, ended); err == nil {
		t.Error("claude error: want an error so the job retries")
	}
	if _, err := db.New(pool).GetBriefingBySession(context.Background(), ended.ID); err == nil {
		t.Error("briefing written although Claude failed")
	}
}
