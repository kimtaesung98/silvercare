// Package session is the conversation engine: it turns one elder utterance
// into an AI turn (sentence 0 from the opener clips, sentences 1..n streamed
// from Claude) and serves the /ws/elder WebSocket that carries it.
// The rules it follows are docs/specs/conversation-loop.md.
package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/escalation"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/llm"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/opener"
)

// Errors HandleElderText returns before saving anything.
var (
	ErrEmptyText       = errors.New("utterance is empty")
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionEnded    = errors.New("session already ended")
)

// Fixed sentences (docs/specs/conversation-loop.md section 2). The reassurance
// is also the text of the ESCALATION opener clip.
const (
	ReassuranceSentence = "지금 많이 힘드시죠. 선생님께 바로 알려드렸어요. 조금만 기다려주세요."
	FallbackSentence    = "네, 말씀 잘 들었어요. 선생님이 곧 오실 거예요. 조금만 더 저랑 이야기 나눠요."
	// No caregiver is on the way in a companion session.
	CompanionFallbackSentence = "네, 말씀 잘 들었어요. 조금 더 이야기해 주세요."
	GreetingFallbackSentence  = "안녕하세요. 저랑 잠깐 이야기 나눠요."
)

// Outcome is how a turn ended (ai.turn_end.outcome).
type Outcome string

// Outcomes.
const (
	Completed Outcome = "COMPLETED"
	Fallback  Outcome = "FALLBACK"
	Escalated Outcome = "ESCALATED"
	Cancelled Outcome = "CANCELLED"
)

// Sink receives a turn's events in order, as they happen. The WebSocket
// connection implements it; calls must not block for long.
type Sink interface {
	Transcript(u db.Utterance)
	Opener(u db.Utterance, clip db.OpenerClip)
	Filler(sessionID uuid.UUID, turn int32, clip db.OpenerClip)
	Reply(u db.Utterance)
	TurnEnd(sessionID uuid.UUID, turn int32, o Outcome, sentences int)
}

// Config are the engine's timings.
type Config struct {
	// FillerAfter plays one filler clip when sentence 1 is not ready by then.
	FillerAfter time.Duration
	// FirstSentenceTimeout gives up on Claude when sentence 1 is not ready by
	// then, and says the fallback sentence instead.
	FirstSentenceTimeout time.Duration
	// TurnTimeout cuts a turn that is still streaming.
	TurnTimeout time.Duration
	// MaxSentences cuts a reply that runs long (the prompt asks for 1 to 3).
	MaxSentences int
}

// DefaultConfig are the timings used unless configured otherwise.
var DefaultConfig = Config{
	FillerAfter:          2 * time.Second,
	FirstSentenceTimeout: 6 * time.Second,
	TurnTimeout:          20 * time.Second,
	MaxSentences:         4,
}

// TurnResult is what one elder utterance produced.
type TurnResult struct {
	Elder db.Utterance
	// Turn is the AI turn's seq (Elder.Seq + 1).
	Turn int32
	// Opener is sentence 0, nil when no clip was available or the turn was
	// cancelled before it.
	Opener  *db.OpenerClip
	Replies []db.Utterance
	// Text is everything the AI said this turn, sentence 0 included.
	Text      string
	Escalated bool
	Trigger   *escalation.TriggerType
	Outcome   Outcome
}

// Alerter hears about every escalation event the engine records, so the
// people who must act are told (internal/jobs queues the push).
type Alerter interface {
	EscalationRaised(ctx context.Context, ev db.EscalationEvent)
}

type nopAlerter struct{}

func (nopAlerter) EscalationRaised(context.Context, db.EscalationEvent) {}

// Engine runs conversation turns.
type Engine struct {
	q       *db.Queries
	llm     llm.Client
	openers *opener.Library
	cfg     Config
	logger  *slog.Logger
	alerter Alerter

	mu       sync.Mutex
	sessions map[uuid.UUID]*sessionState
}

// sessionState serializes the turns of one session. A newer utterance or a
// barge-in cancels the turn in flight.
type sessionState struct {
	turnMu sync.Mutex // held while a turn runs

	mu       sync.Mutex
	ticket   uint64 // latest turn to arrive
	cancel   context.CancelFunc
	turn     int32 // AI turn of ticket, -1 until known
	lastClip uuid.UUID
}

// NewEngine returns an Engine.
func NewEngine(q *db.Queries, client llm.Client, openers *opener.Library, cfg Config, logger *slog.Logger) *Engine {
	return &Engine{q: q, llm: client, openers: openers, cfg: cfg, logger: logger, alerter: nopAlerter{},
		sessions: map[uuid.UUID]*sessionState{}}
}

// SetAlerter sets who hears about escalation events. Call it before serving.
func (e *Engine) SetAlerter(a Alerter) { e.alerter = a }

func (e *Engine) state(sessionID uuid.UUID) *sessionState {
	e.mu.Lock()
	defer e.mu.Unlock()
	st, ok := e.sessions[sessionID]
	if !ok {
		st = &sessionState{turn: -1}
		e.sessions[sessionID] = st
	}
	return st
}

// begin cancels the turn in flight and returns the new turn's context.
func (st *sessionState) begin(ctx context.Context) (context.Context, uint64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.cancel != nil {
		st.cancel()
	}
	tctx, cancel := context.WithCancel(ctx)
	st.ticket++
	st.cancel, st.turn = cancel, -1
	return tctx, st.ticket
}

func (st *sessionState) setTurn(ticket uint64, turn int32) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.ticket == ticket {
		st.turn = turn
	}
}

func (st *sessionState) finish(ticket uint64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.ticket == ticket {
		st.cancel()
		st.cancel, st.turn = nil, -1
	}
}

// BargeIn stops the given AI turn if it is still running: the elder started
// talking over it.
func (e *Engine) BargeIn(sessionID uuid.UUID, turn int32) {
	st := e.state(sessionID)
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.cancel != nil && st.turn == turn {
		st.cancel()
	}
}

// Forget stops any turn of an ended session and drops its state.
func (e *Engine) Forget(sessionID uuid.UUID) {
	e.mu.Lock()
	st, ok := e.sessions[sessionID]
	delete(e.sessions, sessionID)
	e.mu.Unlock()
	if ok {
		st.mu.Lock()
		if st.cancel != nil {
			st.cancel()
		}
		st.mu.Unlock()
	}
}

func (e *Engine) openSession(ctx context.Context, id uuid.UUID) (db.ConversationSession, error) {
	sess, err := e.q.GetSession(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return sess, ErrSessionNotFound
	}
	if err != nil {
		return sess, fmt.Errorf("read session: %w", err)
	}
	if sess.EndedAt != nil {
		return sess, ErrSessionEnded
	}
	return sess, nil
}

// HandleElderText processes one elder utterance (docs/specs/conversation-loop.md
// section 1) and reports its events to sink as they happen. Errors other than
// the three above are database errors; a failing Claude never fails the turn.
func (e *Engine) HandleElderText(ctx context.Context, sessionID uuid.UUID, text string, sink Sink) (TurnResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return TurnResult{}, ErrEmptyText
	}
	st := e.state(sessionID)
	tctx, ticket := st.begin(ctx)
	defer st.finish(ticket)
	st.turnMu.Lock()
	defer st.turnMu.Unlock()

	// Writes outlive a cancelled turn: what was said is always recorded.
	wctx := context.WithoutCancel(ctx)
	sess, err := e.openSession(wctx, sessionID)
	if err != nil {
		return TurnResult{}, err
	}

	// The rules always run first, whatever happens next.
	match, flagged := escalation.Check(text)

	seq, err := e.q.NextUtteranceSeq(wctx, sessionID)
	if err != nil {
		return TurnResult{}, fmt.Errorf("next seq: %w", err)
	}
	elderUtt, err := e.q.CreateUtterance(wctx, db.CreateUtteranceParams{
		SessionID: sessionID, Seq: seq, Speaker: "ELDER", Text: text, FlaggedRisk: flagged,
	})
	if err != nil {
		return TurnResult{}, fmt.Errorf("save elder utterance: %w", err)
	}
	sink.Transcript(elderUtt)

	res := TurnResult{Elder: elderUtt, Turn: seq + 1}
	st.setTurn(ticket, res.Turn)
	t := &turn{e: e, sess: sess, seq: res.Turn, sink: sink, ctx: wctx, start: time.Now()}

	elder, err := e.q.GetElder(wctx, sess.ElderID)
	if err != nil {
		return res, fmt.Errorf("read elder: %w", err)
	}
	manifest, err := e.openers.Manifest(wctx, opener.VoiceOf(elder))
	if err != nil {
		return res, err
	}

	if flagged {
		res.Escalated, res.Trigger = true, &match.Type
		if ev, err := escalation.FromRule(wctx, e.q, sessionID, elderUtt.ID, match); err != nil {
			// The elder still hears the reassurance; the missing event is an
			// alert for us (spec section 5, decided in stage 3).
			e.logger.ErrorContext(ctx, "save escalation event failed",
				"session_id", sessionID, "utterance_id", elderUtt.ID, "rule_id", match.RuleID, "err", err)
		} else {
			e.alerter.EscalationRaised(wctx, ev)
		}
		e.logger.WarnContext(ctx, "escalation", "session_id", sessionID, "rule_id", match.RuleID, "type", match.Type)
		if err := t.reassure(manifest); err != nil {
			return res, err
		}
		return t.result(res, Escalated), nil
	}

	if tctx.Err() != nil {
		// A newer utterance arrived while this one was being saved.
		sink.TurnEnd(sessionID, res.Turn, Cancelled, 0)
		return t.result(res, Cancelled), nil
	}

	clip, err := manifest.Pick(opener.Classify(text), st.lastClip)
	if err != nil {
		e.logger.WarnContext(ctx, "no opener clip", "session_id", sessionID, "err", err)
	} else {
		st.lastClip = clip.ID
		if err := t.opener(clip); err != nil {
			return res, err
		}
	}

	req, err := e.request(wctx, sess, elder)
	if err != nil {
		return res, err
	}
	if t.openerClip != nil {
		req.Continuation = llm.Continuation(t.openerClip.Text)
	}
	outcome, concerns, err := t.generate(tctx, req, manifest, fallbackFor(sess.Mode))
	if err != nil {
		return res, err
	}
	for _, c := range concerns {
		ev, err := escalation.FromLLM(wctx, e.q, sessionID, elderUtt.ID, escalation.TriggerType(c.Type), c.Reason)
		if err != nil {
			e.logger.ErrorContext(ctx, "save llm escalation event failed", "session_id", sessionID, "err", err)
			continue
		}
		e.alerter.EscalationRaised(wctx, ev)
		e.logger.WarnContext(ctx, "llm concern", "session_id", sessionID, "type", c.Type, "reason", c.Reason)
	}
	return t.result(res, outcome), nil
}

// Greet says the AI's opening line (turn 0) of a session that has none yet.
// It returns false when the session already has utterances.
func (e *Engine) Greet(ctx context.Context, sessionID uuid.UUID, sink Sink) (bool, error) {
	st := e.state(sessionID)
	tctx, ticket := st.begin(ctx)
	defer st.finish(ticket)
	st.turnMu.Lock()
	defer st.turnMu.Unlock()

	wctx := context.WithoutCancel(ctx)
	sess, err := e.openSession(wctx, sessionID)
	if err != nil {
		return false, err
	}
	seq, err := e.q.NextUtteranceSeq(wctx, sessionID)
	if err != nil {
		return false, fmt.Errorf("next seq: %w", err)
	}
	if seq != 0 {
		return false, nil
	}
	st.setTurn(ticket, 0)
	elder, err := e.q.GetElder(wctx, sess.ElderID)
	if err != nil {
		return false, fmt.Errorf("read elder: %w", err)
	}
	manifest, err := e.openers.Manifest(wctx, opener.VoiceOf(elder))
	if err != nil {
		return false, err
	}
	req, err := e.request(wctx, sess, elder)
	if err != nil {
		return false, err
	}
	t := &turn{e: e, sess: sess, seq: 0, sink: sink, ctx: wctx, start: time.Now()}
	if _, _, err := t.generate(tctx, req, manifest, GreetingFallbackSentence); err != nil {
		return true, err
	}
	return true, nil
}

func fallbackFor(mode string) string {
	if mode == "COMPANION" {
		return CompanionFallbackSentence
	}
	return FallbackSentence
}

// request builds the Claude request from the session's saved utterances.
func (e *Engine) request(ctx context.Context, sess db.ConversationSession, elder db.Elder) (llm.Request, error) {
	keywords, err := e.q.ListTopKeywords(ctx, db.ListTopKeywordsParams{ElderID: elder.ID, MaxCount: 5})
	if err != nil {
		return llm.Request{}, fmt.Errorf("read keywords: %w", err)
	}
	words := make([]string, len(keywords))
	for i, k := range keywords {
		words[i] = k.Keyword
	}
	history, err := e.q.ListSessionUtterances(ctx, sess.ID)
	if err != nil {
		return llm.Request{}, fmt.Errorf("read utterances: %w", err)
	}
	return llm.Request{
		System:   llm.SystemPrompt(sess.Mode, elder.Name, words),
		Messages: buildMessages(history),
	}, nil
}

// buildMessages turns the saved utterances into Messages API turns (spec
// section 1 step 6): the kickoff instruction first, ELDER as user and AI as
// assistant, consecutive utterances of one speaker merged with a newline.
// The sentences of one AI turn are joined with a space. An AI turn still
// being produced (the last one) is left out: its sentence 0 travels as the
// continuation note instead.
func buildMessages(us []db.Utterance) []llm.Message {
	if n := len(us); n > 0 && us[n-1].Speaker == "AI" {
		cur := us[n-1].Seq
		for n > 0 && us[n-1].Seq == cur {
			n--
		}
		us = us[:n]
	}
	msgs := []llm.Message{{Role: llm.User, Text: llm.Kickoff()}}
	var prevSeq int32 = -1
	for _, u := range us {
		role := llm.User
		if u.Speaker == "AI" {
			role = llm.Assistant
		}
		last := &msgs[len(msgs)-1]
		switch {
		case last.Role != role:
			msgs = append(msgs, llm.Message{Role: role, Text: u.Text})
		case role == llm.Assistant && u.Seq == prevSeq:
			last.Text += " " + u.Text
		default:
			last.Text += "\n" + u.Text
		}
		prevSeq = u.Seq
	}
	return msgs
}

// turn is one AI turn being produced.
type turn struct {
	e     *Engine
	sess  db.ConversationSession
	seq   int32
	sink  Sink
	ctx   context.Context // for writes; never cancelled
	start time.Time

	openerClip *db.OpenerClip
	saved      []db.Utterance // every AI chunk, sentence 0 included
	replies    []db.Utterance
}

func (t *turn) latency() *int32 {
	ms := int32(time.Since(t.start).Milliseconds())
	return &ms
}

func (t *turn) opener(clip db.OpenerClip) error {
	u, err := t.e.q.CreateUtterance(t.ctx, db.CreateUtteranceParams{
		SessionID: t.sess.ID, Seq: t.seq, ChunkIndex: 0, Speaker: "AI", Text: clip.Text,
		OpenerClipID: &clip.ID, LatencyMs: t.latency(),
	})
	if err != nil {
		return fmt.Errorf("save opener: %w", err)
	}
	t.openerClip = &clip
	t.saved = append(t.saved, u)
	t.sink.Opener(u, clip)
	return nil
}

func (t *turn) reply(text string) error {
	u, err := t.e.q.CreateUtterance(t.ctx, db.CreateUtteranceParams{
		SessionID: t.sess.ID, Seq: t.seq, ChunkIndex: int32(len(t.replies) + 1), Speaker: "AI", Text: text,
		LatencyMs: t.latency(),
	})
	if err != nil {
		return fmt.Errorf("save reply: %w", err)
	}
	t.saved = append(t.saved, u)
	t.replies = append(t.replies, u)
	t.sink.Reply(u)
	return nil
}

// reassure answers a rule match: the ESCALATION clip, or the same sentence as
// a reply if no clip is recorded. Claude is not called.
func (t *turn) reassure(m opener.Manifest) error {
	clip, err := m.Pick(opener.Escalation, uuid.Nil)
	if err == nil {
		err = t.opener(clip)
	} else {
		t.e.logger.WarnContext(t.ctx, "no escalation clip", "session_id", t.sess.ID, "err", err)
		err = t.reply(ReassuranceSentence)
	}
	if err != nil {
		return err
	}
	t.sink.TurnEnd(t.sess.ID, t.seq, Escalated, len(t.replies))
	return nil
}

func (t *turn) result(res TurnResult, o Outcome) TurnResult {
	res.Opener = t.openerClip
	res.Replies = t.replies
	parts := make([]string, len(t.saved))
	for i, u := range t.saved {
		parts[i] = u.Text
	}
	res.Text = strings.Join(parts, " ")
	res.Outcome = o
	return res
}

type streamDone struct {
	res llm.Result
	err error
}

// generate streams Claude's reply as sentences 1..n, with the filler, the
// timeouts and the fallback sentence around it, then ends the turn.
// ctx is the turn's context: cancelling it is a barge-in.
func (t *turn) generate(ctx context.Context, req llm.Request, m opener.Manifest, fallback string) (Outcome, []llm.Concern, error) {
	cfg := t.e.cfg
	sctx, stop := context.WithCancel(ctx)
	defer stop()

	deltas := make(chan string, 256)
	done := make(chan streamDone, 1)
	go func() {
		r, err := t.e.llm.Stream(sctx, req, func(s string) {
			select {
			case deltas <- s:
			case <-sctx.Done():
			}
		})
		done <- streamDone{r, err}
	}()

	filler := time.NewTimer(cfg.FillerAfter)
	first := time.NewTimer(cfg.FirstSentenceTimeout)
	overall := time.NewTimer(cfg.TurnTimeout)
	defer filler.Stop()
	defer first.Stop()
	defer overall.Stop()

	var (
		sp       splitter
		outcome  = Completed
		finished *streamDone
		cut      bool
	)
	emit := func(sentences []string) error {
		for _, s := range sentences {
			if len(t.replies) >= cfg.MaxSentences {
				cut = true
				return nil
			}
			if err := t.reply(s); err != nil {
				return err
			}
		}
		return nil
	}

loop:
	for !cut {
		select {
		case d := <-deltas:
			if err := emit(sp.Push(d)); err != nil {
				return "", nil, err
			}
		case d := <-done:
			// Stream calls onText before returning, so every delta is buffered.
			for more := true; more; {
				select {
				case s := <-deltas:
					if err := emit(sp.Push(s)); err != nil {
						return "", nil, err
					}
				default:
					more = false
				}
			}
			if rest := sp.Flush(); rest != "" && d.err == nil && !d.res.Refused {
				if err := emit([]string{rest}); err != nil {
					return "", nil, err
				}
			}
			finished = &d
			break loop
		case <-filler.C:
			if len(t.replies) == 0 {
				if clip, err := m.Pick(opener.Filler, uuid.Nil); err == nil {
					t.sink.Filler(t.sess.ID, t.seq, clip)
				}
			}
		case <-first.C:
			if len(t.replies) == 0 {
				t.e.logger.WarnContext(t.ctx, "claude first sentence timeout", "session_id", t.sess.ID, "turn", t.seq)
				break loop
			}
		case <-overall.C:
			t.e.logger.WarnContext(t.ctx, "claude turn timeout", "session_id", t.sess.ID, "turn", t.seq)
			break loop
		case <-ctx.Done():
			outcome = Cancelled
			break loop
		}
	}
	stop()

	var concerns []llm.Concern
	if finished != nil {
		switch {
		case finished.err != nil && ctx.Err() == nil:
			t.e.logger.ErrorContext(t.ctx, "claude call failed", "session_id", t.sess.ID, "turn", t.seq, "err", finished.err)
		case finished.res.Refused:
			t.e.logger.WarnContext(t.ctx, "claude refused", "session_id", t.sess.ID, "turn", t.seq)
		}
		concerns = finished.res.Concerns
	}
	if outcome != Cancelled && len(t.replies) == 0 {
		outcome = Fallback
		if err := t.reply(fallback); err != nil {
			return "", nil, err
		}
	}
	if finished != nil && finished.err == nil && len(t.saved) > 0 {
		last := t.saved[len(t.saved)-1]
		in, out := int32(finished.res.InputTokens), int32(finished.res.OutputTokens)
		model := finished.res.Model
		if err := t.e.q.SetUtteranceUsage(t.ctx, db.SetUtteranceUsageParams{
			ID: last.ID, Model: &model, InputTokens: &in, OutputTokens: &out,
		}); err != nil {
			t.e.logger.ErrorContext(t.ctx, "save token usage failed", "session_id", t.sess.ID, "err", err)
		}
	}
	t.sink.TurnEnd(t.sess.ID, t.seq, outcome, len(t.replies))
	return outcome, concerns, nil
}
