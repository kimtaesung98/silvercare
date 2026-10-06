package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/escalation"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/llm"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/opener"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/testdb"
)

// fakeLLM records requests and streams a canned reply.
type fakeLLM struct {
	mu       sync.Mutex
	reqs     []llm.Request
	chunks   []string
	err      error
	concerns []llm.Concern
	block    bool // wait for the context to end, like a stalled stream
}

func (f *fakeLLM) Stream(ctx context.Context, req llm.Request, onText func(string)) (llm.Result, error) {
	f.mu.Lock()
	f.reqs = append(f.reqs, req)
	block, err, chunks, concerns := f.block, f.err, f.chunks, f.concerns
	f.mu.Unlock()
	if block {
		<-ctx.Done()
		return llm.Result{}, ctx.Err()
	}
	if err != nil {
		return llm.Result{}, err
	}
	for _, c := range chunks {
		onText(c)
	}
	return llm.Result{Model: "fake-model", InputTokens: 100, OutputTokens: 20, Concerns: concerns}, nil
}

func (f *fakeLLM) requests() []llm.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.reqs)
}

// recSink records events as short strings.
type recSink struct {
	mu     sync.Mutex
	events []string
	opened chan struct{}
}

func newRecSink() *recSink { return &recSink{opened: make(chan struct{}, 8)} }

func (s *recSink) add(e string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
}

func (s *recSink) Transcript(u db.Utterance) { s.add(fmt.Sprintf("transcript %d %s", u.Seq, u.Text)) }
func (s *recSink) Opener(u db.Utterance, c db.OpenerClip) {
	s.add(fmt.Sprintf("opener %d %s", u.Seq, c.Category))
	s.opened <- struct{}{}
}
func (s *recSink) Filler(_ uuid.UUID, turn int32, _ db.OpenerClip) {
	s.add(fmt.Sprintf("filler %d", turn))
}
func (s *recSink) Reply(u db.Utterance) {
	s.add(fmt.Sprintf("reply %d.%d %s", u.Seq, u.ChunkIndex, u.Text))
}
func (s *recSink) TurnEnd(_ uuid.UUID, turn int32, o Outcome, n int) {
	s.add(fmt.Sprintf("end %d %s %d", turn, o, n))
}

func (s *recSink) list() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.events)
}

type env struct {
	pool    *pgxpool.Pool
	q       *db.Queries
	llm     *fakeLLM
	engine  *Engine
	elderID uuid.UUID
	session uuid.UUID
}

func newEnv(t *testing.T, cfg Config) *env {
	t.Helper()
	pool := testdb.New(t)
	f := testdb.Seed(t, pool, time.Now().Add(time.Hour))
	q := db.New(pool)
	ctx := context.Background()
	sess, err := q.CreatePickupSession(ctx, db.CreatePickupSessionParams{
		VisitID: &f.VisitID, ElderID: f.ElderID, TriggerEtaMinutes: ptr(int32(12)),
	})
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeLLM{chunks: []string{"손주분이 오셨다니 ", "좋으셨겠어요. 무엇을 ", "하고 노셨어요?"}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &env{
		pool: pool, q: q, llm: fake,
		engine:  NewEngine(q, fake, opener.NewLibrary(q), cfg, logger),
		elderID: f.ElderID, session: sess.ID,
	}
}

func (e *env) say(t *testing.T, seq int32, speaker, text string) {
	t.Helper()
	if _, err := e.q.CreateUtterance(context.Background(), db.CreateUtteranceParams{
		SessionID: e.session, Seq: seq, Speaker: speaker, Text: text,
	}); err != nil {
		t.Fatal(err)
	}
}

func (e *env) utterances(t *testing.T) []db.Utterance {
	t.Helper()
	us, err := e.q.ListSessionUtterances(context.Background(), e.session)
	if err != nil {
		t.Fatal(err)
	}
	return us
}

func (e *env) escalations(t *testing.T) []db.EscalationEvent {
	t.Helper()
	rows, err := e.pool.Query(context.Background(),
		`SELECT id, session_id, utterance_id, trigger_type, source, rule_id, reason FROM escalation_event WHERE session_id = $1 ORDER BY created_at`, e.session)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []db.EscalationEvent
	for rows.Next() {
		var ev db.EscalationEvent
		if err := rows.Scan(&ev.ID, &ev.SessionID, &ev.UtteranceID, &ev.TriggerType, &ev.Source, &ev.RuleID, &ev.Reason); err != nil {
			t.Fatal(err)
		}
		out = append(out, ev)
	}
	return out
}

func roles(msgs []llm.Message) []llm.Role {
	out := make([]llm.Role, len(msgs))
	for i, m := range msgs {
		out[i] = m.Role
	}
	return out
}

// Spec 4.2 case 1: an ordinary utterance.
func TestOrdinaryUtterance(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	ctx := context.Background()
	if _, err := e.q.BumpKeyword(ctx, db.BumpKeywordParams{
		ElderID: e.elderID, Keyword: "손주", Category: "FAMILY", Mentions: 3, EmotionTone: "POSITIVE", MentionedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	e.say(t, 0, "AI", "안녕하세요, 김순자님.")
	sink := newRecSink()

	res, err := e.engine.HandleElderText(ctx, e.session, "  손주가 어제 왔어 ", sink)
	if err != nil {
		t.Fatal(err)
	}

	if res.Elder.Text != "손주가 어제 왔어" || res.Elder.FlaggedRisk || res.Elder.Seq != 1 {
		t.Errorf("elder utterance = %+v", res.Elder)
	}
	if res.Escalated || res.Trigger != nil || res.Outcome != Completed {
		t.Errorf("result = %+v", res)
	}
	if res.Opener == nil || res.Opener.Category != string(opener.Recall) {
		t.Fatalf("opener = %+v", res.Opener)
	}

	reqs := e.llm.requests()
	if len(reqs) != 1 {
		t.Fatalf("claude calls = %d", len(reqs))
	}
	req := reqs[0]
	system := strings.Join(req.System, "\n")
	for _, want := range []string{"김순자", "손주"} {
		if !strings.Contains(system, want) {
			t.Errorf("system prompt lacks %q", want)
		}
	}
	if got := roles(req.Messages); !slices.Equal(got, []llm.Role{llm.User, llm.Assistant, llm.User}) {
		t.Errorf("roles = %v", got)
	}
	if got := req.Messages[len(req.Messages)-1].Text; got != "손주가 어제 왔어" {
		t.Errorf("last message = %q", got)
	}
	if !strings.Contains(req.Continuation, res.Opener.Text) {
		t.Errorf("continuation %q lacks opener %q", req.Continuation, res.Opener.Text)
	}

	wantReplies := []string{"손주분이 오셨다니 좋으셨겠어요.", "무엇을 하고 노셨어요?"}
	var got []string
	for _, r := range res.Replies {
		got = append(got, r.Text)
	}
	if !slices.Equal(got, wantReplies) {
		t.Errorf("replies = %q", got)
	}
	if want := res.Opener.Text + " " + strings.Join(wantReplies, " "); res.Text != want {
		t.Errorf("text = %q, want %q", res.Text, want)
	}

	us := e.utterances(t)
	if len(us) != 5 { // greeting, elder, opener, 2 replies
		t.Fatalf("utterances = %d", len(us))
	}
	for i, u := range us[2:] {
		if u.Speaker != "AI" || u.Seq != 2 || u.ChunkIndex != int32(i) {
			t.Errorf("AI chunk %d = seq %d chunk %d %s", i, u.Seq, u.ChunkIndex, u.Speaker)
		}
	}
	if us[2].OpenerClipID == nil || *us[2].OpenerClipID != res.Opener.ID {
		t.Errorf("opener utterance clip = %v", us[2].OpenerClipID)
	}
	last := us[4]
	if last.Model == nil || *last.Model != "fake-model" || last.InputTokens == nil || *last.InputTokens != 100 || *last.OutputTokens != 20 {
		t.Errorf("usage on last chunk = %v %v %v", last.Model, last.InputTokens, last.OutputTokens)
	}
	if evs := e.escalations(t); len(evs) != 0 {
		t.Errorf("escalations = %d", len(evs))
	}

	wantEvents := []string{
		"transcript 1 손주가 어제 왔어",
		"opener 2 RECALL",
		"reply 2.1 손주분이 오셨다니 좋으셨겠어요.",
		"reply 2.2 무엇을 하고 노셨어요?",
		"end 2 COMPLETED 2",
	}
	if got := sink.list(); !slices.Equal(got, wantEvents) {
		t.Errorf("events = %q", got)
	}
}

// Spec 4.2 case 2 and 4.1: a rule match never reaches Claude.
func TestEscalation(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	e.say(t, 0, "AI", "안녕하세요.")
	sink := newRecSink()

	res, err := e.engine.HandleElderText(context.Background(), e.session, "화장실에서 넘어졌어", sink)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Elder.FlaggedRisk {
		t.Error("elder utterance not flagged")
	}
	if !res.Escalated || res.Trigger == nil || *res.Trigger != escalation.FallMention || res.Outcome != Escalated {
		t.Errorf("result = %+v", res)
	}
	if res.Text != ReassuranceSentence {
		t.Errorf("text = %q", res.Text)
	}
	if n := len(e.llm.requests()); n != 0 {
		t.Errorf("claude called %d times", n)
	}
	evs := e.escalations(t)
	if len(evs) != 1 {
		t.Fatalf("escalations = %d", len(evs))
	}
	ev := evs[0]
	if ev.SessionID != e.session || ev.UtteranceID == nil || *ev.UtteranceID != res.Elder.ID ||
		ev.TriggerType != "FALL_MENTION" || ev.Source != "RULE" || ev.RuleID == nil || *ev.RuleID != "fall.fell" {
		t.Errorf("event = %+v", ev)
	}
	want := []string{"transcript 1 화장실에서 넘어졌어", "opener 2 ESCALATION", "end 2 ESCALATED 0"}
	if got := sink.list(); !slices.Equal(got, want) {
		t.Errorf("events = %q", got)
	}
}

// Every 4.1 detect sentence escalates with its type; the others reach Claude.
func TestRuleTable(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	for _, tc := range []struct {
		text string
		want escalation.TriggerType
	}{
		{"아까 화장실에서 넘어졌어", escalation.FallMention},
		{"길에서 쓰러졌었지", escalation.FallMention},
		{"다리가 너무 아파", escalation.PainComplaint},
		{"머리가 아프네", escalation.PainComplaint},
		{"숨이 차서 힘들어", escalation.PainComplaint},
		{"그냥 죽고 싶어", escalation.SelfOrOtherHarm},
		{"누가 나를 때리려고 해", escalation.SelfOrOtherHarm},
		{"오늘 날씨가 좋네", ""},
		{"손주가 어제 놀러 왔어", ""},
		{"우리 아파트 앞에 꽃이 폈어", ""},
	} {
		res, err := e.engine.HandleElderText(context.Background(), e.session, tc.text, newRecSink())
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case tc.want == "" && res.Escalated:
			t.Errorf("%q escalated as %s", tc.text, *res.Trigger)
		case tc.want != "" && (!res.Escalated || *res.Trigger != tc.want):
			t.Errorf("%q: escalated=%v trigger=%v, want %s", tc.text, res.Escalated, res.Trigger, tc.want)
		}
	}
	if got := len(e.escalations(t)); got != 7 {
		t.Errorf("escalation events = %d, want 7", got)
	}
	if got := len(e.llm.requests()); got != 3 {
		t.Errorf("claude calls = %d, want 3", got)
	}
}

// Spec 4.2 case 3: consecutive elder utterances merge into one user turn.
func TestConsecutiveElderUtterancesMerge(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	e.say(t, 0, "AI", "안녕하세요, 김순자님.")
	e.say(t, 1, "ELDER", "응")

	if _, err := e.engine.HandleElderText(context.Background(), e.session, "오늘 날씨 좋네", newRecSink()); err != nil {
		t.Fatal(err)
	}
	msgs := e.llm.requests()[0].Messages
	if len(msgs) != 3 {
		t.Fatalf("messages = %+v", msgs)
	}
	if last := msgs[2]; last.Role != llm.User || last.Text != "응\n오늘 날씨 좋네" {
		t.Errorf("last = %+v", last)
	}
}

// The sentences of an earlier AI turn go back to Claude as one assistant message.
func TestBuildMessagesJoinsChunks(t *testing.T) {
	us := []db.Utterance{
		{Seq: 0, ChunkIndex: 1, Speaker: "AI", Text: "안녕하세요."},
		{Seq: 0, ChunkIndex: 2, Speaker: "AI", Text: "오늘 기분은 어떠세요?"},
		{Seq: 1, Speaker: "ELDER", Text: "좋아"},
		{Seq: 2, ChunkIndex: 0, Speaker: "AI", Text: "그럼요."},
		{Seq: 2, ChunkIndex: 1, Speaker: "AI", Text: "다행이에요."},
		{Seq: 3, Speaker: "ELDER", Text: "손주가 왔어"},
		{Seq: 4, ChunkIndex: 0, Speaker: "AI", Text: "아이고, 그러셨어요."}, // turn in progress
	}
	got := buildMessages(us)
	want := []llm.Message{
		{Role: llm.User, Text: llm.Kickoff()},
		{Role: llm.Assistant, Text: "안녕하세요. 오늘 기분은 어떠세요?"},
		{Role: llm.User, Text: "좋아"},
		{Role: llm.Assistant, Text: "그럼요. 다행이에요."},
		{Role: llm.User, Text: "손주가 왔어"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v", got)
	}
}

// Spec 4.2 cases 4 and 5: Claude failing or saying nothing.
func TestFallback(t *testing.T) {
	for name, set := range map[string]func(*fakeLLM){
		"error": func(f *fakeLLM) { f.err = errors.New("overloaded") },
		"empty": func(f *fakeLLM) { f.chunks = nil },
		"blank": func(f *fakeLLM) { f.chunks = []string{"  ", "\n"} },
	} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t, DefaultConfig)
			e.say(t, 0, "AI", "안녕하세요.")
			set(e.llm)
			sink := newRecSink()
			res, err := e.engine.HandleElderText(context.Background(), e.session, "손주가 어제 왔어", sink)
			if err != nil {
				t.Fatal(err)
			}
			if res.Escalated || res.Outcome != Fallback {
				t.Errorf("result = %+v", res)
			}
			if len(res.Replies) != 1 || res.Replies[0].Text != FallbackSentence || res.Replies[0].ChunkIndex != 1 {
				t.Errorf("replies = %+v", res.Replies)
			}
			if !strings.HasSuffix(res.Text, FallbackSentence) {
				t.Errorf("text = %q", res.Text)
			}
			if got := sink.list(); got[len(got)-1] != "end 2 FALLBACK 1" {
				t.Errorf("events = %q", got)
			}
		})
	}
}

func TestCompanionFallback(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	ctx := context.Background()
	if _, err := e.q.EndSession(ctx, db.EndSessionParams{ID: e.session, EndedReason: ptr("CAREGIVER_ARRIVED")}); err != nil {
		t.Fatal(err)
	}
	sess, err := e.q.CreateCompanionSession(ctx, db.CreateCompanionSessionParams{ElderID: e.elderID, StartedBy: "ELDER"})
	if err != nil {
		t.Fatal(err)
	}
	e.llm.err = errors.New("down")
	res, err := e.engine.HandleElderText(ctx, sess.ID, "심심하네", newRecSink())
	if err != nil {
		t.Fatal(err)
	}
	if res.Replies[0].Text != CompanionFallbackSentence {
		t.Errorf("reply = %q", res.Replies[0].Text)
	}
	if !strings.Contains(e.llm.requests()[0].System[0], "말동무") {
		t.Error("companion system prompt not used")
	}
}

// Spec 4.2 cases 6, 7 and 8: rejected before anything is saved.
func TestRejected(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	ctx := context.Background()

	if _, err := e.engine.HandleElderText(ctx, e.session, " \t\n ", newRecSink()); !errors.Is(err, ErrEmptyText) {
		t.Errorf("blank: err = %v", err)
	}
	if _, err := e.engine.HandleElderText(ctx, uuid.New(), "안녕", newRecSink()); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("unknown session: err = %v", err)
	}
	if _, err := e.q.EndSession(ctx, db.EndSessionParams{ID: e.session, EndedReason: ptr("CAREGIVER_ARRIVED")}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.engine.HandleElderText(ctx, e.session, "안녕", newRecSink()); !errors.Is(err, ErrSessionEnded) {
		t.Errorf("ended session: err = %v", err)
	}
	if us := e.utterances(t); len(us) != 0 {
		t.Errorf("utterances saved: %d", len(us))
	}
	if n := len(e.llm.requests()); n != 0 {
		t.Errorf("claude called %d times", n)
	}
}

// flag_concern raises an LLM event next to the reply.
func TestFlagConcern(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	e.llm.concerns = []llm.Concern{{Type: "OTHER_ANOMALY", Reason: "집이 어딘지 모르겠다고 하심"}, {Type: "NONSENSE", Reason: "알 수 없는 유형"}}
	res, err := e.engine.HandleElderText(context.Background(), e.session, "여기가 어디야", newRecSink())
	if err != nil {
		t.Fatal(err)
	}
	if res.Escalated {
		t.Error("an LLM concern does not replace the reply")
	}
	evs := e.escalations(t)
	if len(evs) != 2 {
		t.Fatalf("events = %d", len(evs))
	}
	for _, ev := range evs {
		if ev.Source != "LLM" || ev.RuleID != nil || ev.TriggerType != "OTHER_ANOMALY" || ev.Reason == nil ||
			ev.UtteranceID == nil || *ev.UtteranceID != res.Elder.ID {
			t.Errorf("event = %+v", ev)
		}
	}
}

// A barge-in stops the turn without a fallback sentence.
func TestBargeIn(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	e.llm.block = true
	sink := newRecSink()
	done := make(chan TurnResult)
	go func() {
		res, err := e.engine.HandleElderText(context.Background(), e.session, "옛날에 시장에서 떡을 팔았지", sink)
		if err != nil {
			t.Error(err)
		}
		done <- res
	}()
	<-sink.opened
	e.engine.BargeIn(e.session, 0) // another turn: ignored
	select {
	case <-done:
		t.Fatal("turn ended on a barge-in for another turn")
	case <-time.After(50 * time.Millisecond):
	}
	e.engine.BargeIn(e.session, 1)
	res := <-done
	if res.Outcome != Cancelled || len(res.Replies) != 0 {
		t.Errorf("result = %+v", res)
	}
	if got := sink.list(); got[len(got)-1] != "end 1 CANCELLED 0" {
		t.Errorf("events = %q", got)
	}
}

// A new utterance cancels the turn still streaming, then runs.
func TestNewUtteranceCancelsTurnInFlight(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	e.llm.block = true
	first := newRecSink()
	done := make(chan TurnResult)
	go func() {
		res, _ := e.engine.HandleElderText(context.Background(), e.session, "옛날에 시장에서 떡을 팔았지", first)
		done <- res
	}()
	<-first.opened
	e.llm.mu.Lock()
	e.llm.block = false
	e.llm.mu.Unlock()

	res2, err := e.engine.HandleElderText(context.Background(), e.session, "그 떡이 맛있었어", newRecSink())
	if err != nil {
		t.Fatal(err)
	}
	if res1 := <-done; res1.Outcome != Cancelled {
		t.Errorf("first turn = %s", res1.Outcome)
	}
	if res2.Outcome != Completed || res2.Elder.Seq != 2 || res2.Turn != 3 {
		t.Errorf("second turn = %+v", res2)
	}
}

// A slow first sentence plays a filler, then times out into the fallback.
func TestSlowClaude(t *testing.T) {
	cfg := DefaultConfig
	cfg.FillerAfter = 20 * time.Millisecond
	cfg.FirstSentenceTimeout = 80 * time.Millisecond
	e := newEnv(t, cfg)
	e.llm.block = true
	sink := newRecSink()
	res, err := e.engine.HandleElderText(context.Background(), e.session, "옛날에 시장에서 떡을 팔았지", sink)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != Fallback || res.Replies[0].Text != FallbackSentence {
		t.Errorf("result = %+v", res)
	}
	want := []string{
		"transcript 0 옛날에 시장에서 떡을 팔았지", "opener 1 RECALL", "filler 1",
		"reply 1.1 " + FallbackSentence, "end 1 FALLBACK 1",
	}
	if got := sink.list(); !slices.Equal(got, want) {
		t.Errorf("events = %q", got)
	}
}

func TestMaxSentences(t *testing.T) {
	cfg := DefaultConfig
	cfg.MaxSentences = 2
	e := newEnv(t, cfg)
	e.llm.chunks = []string{"하나요. 둘이요. 셋이요. 넷이요."}
	res, err := e.engine.HandleElderText(context.Background(), e.session, "옛날 이야기 해줄게", newRecSink())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Replies) != 2 || res.Outcome != Completed {
		t.Errorf("result = %+v", res)
	}
}

func TestGreet(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	e.llm.chunks = []string{"안녕하세요, 김순자님. 오늘 기분은 어떠세요?"}
	sink := newRecSink()
	greeted, err := e.engine.Greet(context.Background(), e.session, sink)
	if err != nil || !greeted {
		t.Fatalf("greet = %v, %v", greeted, err)
	}
	req := e.llm.requests()[0]
	if len(req.Messages) != 1 || req.Messages[0].Text != llm.Kickoff() || req.Continuation != "" {
		t.Errorf("request = %+v", req)
	}
	want := []string{"reply 0.1 안녕하세요, 김순자님.", "reply 0.2 오늘 기분은 어떠세요?", "end 0 COMPLETED 2"}
	if got := sink.list(); !slices.Equal(got, want) {
		t.Errorf("events = %q", got)
	}
	if greeted, err := e.engine.Greet(context.Background(), e.session, newRecSink()); err != nil || greeted {
		t.Errorf("second greet = %v, %v", greeted, err)
	}

	e2 := newEnv(t, DefaultConfig)
	e2.llm.err = errors.New("down")
	sink2 := newRecSink()
	if _, err := e2.engine.Greet(context.Background(), e2.session, sink2); err != nil {
		t.Fatal(err)
	}
	if got := sink2.list(); !slices.Equal(got, []string{"reply 0.1 " + GreetingFallbackSentence, "end 0 FALLBACK 1"}) {
		t.Errorf("fallback greeting events = %q", got)
	}
}

// The ESCALATION clip in the seed data is the spec's reassurance sentence.
func TestEscalationClipText(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	m, err := opener.NewLibrary(e.q).Manifest(context.Background(), opener.DefaultVoice)
	if err != nil {
		t.Fatal(err)
	}
	clip, err := m.Pick(opener.Escalation, uuid.Nil)
	if err != nil || clip.Text != ReassuranceSentence {
		t.Errorf("clip = %+v, %v", clip, err)
	}
	for _, c := range []opener.Category{opener.Recall, opener.Question, opener.Emotion, opener.ShortAnswer, opener.Greeting, opener.Filler} {
		if _, err := m.Pick(c, uuid.Nil); err != nil {
			t.Errorf("no %s clip: %v", c, err)
		}
	}
}
