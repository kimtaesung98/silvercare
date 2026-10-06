package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/api"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/opener"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/speech"
)

// Connection timings (api/ws-events.md section 1).
const (
	pingEvery   = 30 * time.Second
	readTimeout = 75 * time.Second // two missed pings
	sendBuffer  = 256
	maxFrame    = 4 << 20 // one utterance of audio
)

// HubConfig are the Hub's collaborators besides the database and engine.
type HubConfig struct {
	// PromptVersion is recorded on companion sessions the tablet starts.
	PromptVersion string
	STT           speech.Recognizer
	TTS           speech.Synthesizer
}

// Hub serves GET /ws/elder: one connection per elder tablet. It also tells
// connected tablets about sessions the visit service starts and ends.
type Hub struct {
	q       *db.Queries
	engine  *Engine
	openers *opener.Library
	cfg     HubConfig
	logger  *slog.Logger
	schema  *jsonschema.Schema

	mu       sync.Mutex
	byDevice map[uuid.UUID]*conn
	byElder  map[uuid.UUID]*conn
	lastEta  map[uuid.UUID]int // session → minutes last sent
}

// NewHub returns a Hub.
func NewHub(q *db.Queries, engine *Engine, openers *opener.Library, cfg HubConfig, logger *slog.Logger) (*Hub, error) {
	if cfg.STT == nil {
		cfg.STT = speech.Unavailable{}
	}
	if cfg.TTS == nil {
		cfg.TTS = speech.Unavailable{}
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(api.WSEventsSchema))
	if err != nil {
		return nil, fmt.Errorf("parse ws-events schema: %w", err)
	}
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	if err := c.AddResource("ws-events.schema.json", doc); err != nil {
		return nil, err
	}
	schema, err := c.Compile("ws-events.schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile ws-events schema: %w", err)
	}
	return &Hub{
		q: q, engine: engine, openers: openers, cfg: cfg, logger: logger, schema: schema,
		byDevice: map[uuid.UUID]*conn{}, byElder: map[uuid.UUID]*conn{}, lastEta: map[uuid.UUID]int{},
	}, nil
}

// envelope is a text frame (api/ws-events.md section 2).
type envelope struct {
	Type string          `json:"type"`
	ID   string          `json:"id,omitempty"`
	TS   *time.Time      `json:"ts,omitempty"`
	Data json.RawMessage `json:"data"`
}

// ServeHTTP upgrades an authenticated tablet to a WebSocket.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tok, ok := auth.BearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeUnauthorized(w)
		return
	}
	tablet, err := auth.LookupTablet(r.Context(), h.q, tok)
	if errors.Is(err, auth.ErrUnknownDevice) {
		writeUnauthorized(w)
		return
	}
	if err != nil {
		h.logger.ErrorContext(r.Context(), "ws auth", "err", err)
		http.Error(w, `{"code":"INTERNAL","message":"서버 오류가 발생했습니다."}`, http.StatusInternalServerError)
		return
	}

	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		return // Accept already answered
	}
	ws.SetReadLimit(maxFrame)
	ctx, cancel := context.WithCancel(context.WithoutCancel(r.Context()))
	c := &conn{
		h: h, ws: ws, tablet: tablet, ctx: ctx, cancel: cancel,
		out: make(chan []frame, sendBuffer), events: make(chan func(), sendBuffer),
	}
	h.register(c)
	defer h.unregister(c)
	c.run()
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"code":"UNAUTHENTICATED","message":"인증이 필요합니다."}`))
}

func (h *Hub) register(c *conn) {
	h.mu.Lock()
	old := h.byDevice[c.tablet.DeviceID]
	h.byDevice[c.tablet.DeviceID] = c
	h.byElder[c.tablet.ElderID] = c
	h.mu.Unlock()
	if old != nil {
		old.close(websocket.StatusPolicyViolation, "replaced by a newer connection")
	}
}

func (h *Hub) unregister(c *conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.byDevice[c.tablet.DeviceID] == c {
		delete(h.byDevice, c.tablet.DeviceID)
	}
	if h.byElder[c.tablet.ElderID] == c {
		delete(h.byElder, c.tablet.ElderID)
	}
}

func (h *Hub) connFor(elderID uuid.UUID) *conn {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.byElder[elderID]
}

// SessionStarted implements visit.SessionNotifier.
func (h *Hub) SessionStarted(_ context.Context, s db.ConversationSession, etaMinutes int) {
	h.mu.Lock()
	h.lastEta[s.ID] = etaMinutes
	h.mu.Unlock()
	if c := h.connFor(s.ElderID); c != nil {
		c.started(s, &etaMinutes)
	}
}

// SessionEnded implements visit.SessionNotifier.
func (h *Hub) SessionEnded(_ context.Context, s db.ConversationSession) {
	h.engine.Forget(s.ID)
	h.mu.Lock()
	delete(h.lastEta, s.ID)
	h.mu.Unlock()
	if c := h.connFor(s.ElderID); c != nil && s.EndedReason != nil {
		// After the events of a turn still being delivered.
		reason := *s.EndedReason
		c.later(func() { c.send("session.ended", map[string]any{"sessionId": s.ID, "reason": reason}) })
	}
}

// EtaUpdated implements visit.SessionNotifier. Only changes reach the tablet.
func (h *Hub) EtaUpdated(_ context.Context, s db.ConversationSession, minutes int) {
	h.mu.Lock()
	last, ok := h.lastEta[s.ID]
	h.lastEta[s.ID] = minutes
	h.mu.Unlock()
	if ok && last == minutes {
		return
	}
	if c := h.connFor(s.ElderID); c != nil && s.VisitID != nil {
		c.send("caregiver.eta", map[string]any{"sessionId": s.ID, "visitId": *s.VisitID, "minutes": minutes})
	}
}

// conn is one tablet connection. Only the writer goroutine writes to ws.
type conn struct {
	h      *Hub
	ws     *websocket.Conn
	tablet auth.Tablet
	out    chan []frame // each item is written as one unit
	events chan func()  // a turn's events, run in order (TTS happens here)
	voice  string       // the elder's TTS voice
	ctx    context.Context
	cancel context.CancelFunc
	nonce  atomic.Int64
	closed atomic.Bool
}

func (c *conn) log() *slog.Logger {
	return c.h.logger.With("device_id", c.tablet.DeviceID, "elder_id", c.tablet.ElderID)
}

func (c *conn) close(code websocket.StatusCode, reason string) {
	if c.closed.CompareAndSwap(false, true) {
		c.cancel()
		_ = c.ws.Close(code, reason)
	}
}

func (c *conn) run() {
	defer c.close(websocket.StatusNormalClosure, "")
	go c.writeLoop()
	go c.eventLoop()
	if err := c.ready(); err != nil {
		c.log().Error("connection ready", "err", err)
		c.close(websocket.StatusInternalError, "internal error")
		return
	}
	for {
		rctx, cancel := context.WithTimeout(c.ctx, readTimeout)
		typ, data, err := c.ws.Read(rctx)
		cancel()
		if err != nil {
			if websocket.CloseStatus(err) == -1 && c.ctx.Err() == nil {
				c.log().Info("ws read ended", "err", err)
			}
			return
		}
		if typ != websocket.MessageText {
			c.sendError("INVALID_MESSAGE", "오디오 프레임은 elder.audio 다음에만 보낼 수 있습니다.", "")
			continue
		}
		c.handle(data)
	}
}

func (c *conn) writeLoop() {
	ping := time.NewTicker(pingEvery)
	defer ping.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case frames := <-c.out:
			for _, f := range frames {
				if err := c.ws.Write(c.ctx, f.typ, f.data); err != nil {
					c.close(websocket.StatusInternalError, "write failed")
					return
				}
			}
		case <-ping.C:
			c.send("ping", map[string]any{"nonce": strconv.FormatInt(c.nonce.Add(1), 10)})
		}
	}
}

// frame is one WebSocket message.
type frame struct {
	typ  websocket.MessageType
	data []byte
}

// send queues a text frame.
func (c *conn) send(typ string, data any) {
	if msg, ok := c.encode(typ, data); ok {
		c.enqueue(frame{websocket.MessageText, msg})
	}
}

// sendWithAudio queues a text frame and the binary audio frame that must
// follow it (api/ws-events.md section 2) as one unit.
func (c *conn) sendWithAudio(typ string, data any, audio []byte) {
	if msg, ok := c.encode(typ, data); ok {
		c.enqueue(frame{websocket.MessageText, msg}, frame{websocket.MessageBinary, audio})
	}
}

func (c *conn) encode(typ string, data any) ([]byte, bool) {
	raw, err := json.Marshal(data)
	if err != nil {
		c.log().Error("marshal ws event", "type", typ, "err", err)
		return nil, false
	}
	now := time.Now().UTC()
	msg, err := json.Marshal(envelope{Type: typ, TS: &now, Data: raw})
	if err != nil {
		c.log().Error("marshal ws envelope", "type", typ, "err", err)
		return nil, false
	}
	return msg, true
}

// enqueue drops a tablet too slow to drain its queue; it reconnects and
// resumes from connection.ready.
func (c *conn) enqueue(frames ...frame) {
	if c.closed.Load() {
		return
	}
	select {
	case c.out <- frames:
	default:
		c.log().Warn("ws send buffer full, closing")
		c.close(websocket.StatusTryAgainLater, "too slow")
	}
}

// later runs f on the event goroutine, after the events queued before it.
func (c *conn) later(f func()) {
	if c.closed.Load() {
		return
	}
	select {
	case c.events <- f:
	default:
		c.log().Warn("ws event queue full, closing")
		c.close(websocket.StatusTryAgainLater, "too slow")
	}
}

func (c *conn) eventLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case f := <-c.events:
			f()
		}
	}
}

func (c *conn) sendError(code, message, replyTo string) {
	d := map[string]any{"code": code, "message": message}
	if replyTo != "" {
		d["replyTo"] = replyTo
	}
	c.send("error", d)
}

// ready sends connection.ready and greets an open session that has not
// started talking yet (the tablet was offline when it started).
func (c *conn) ready() error {
	elder, err := c.h.q.GetElder(c.ctx, c.tablet.ElderID)
	if err != nil {
		return fmt.Errorf("read elder: %w", err)
	}
	c.voice = opener.VoiceOf(elder)
	m, err := c.h.openers.Manifest(c.ctx, c.voice)
	if err != nil {
		return err
	}
	var active any
	sess, err := c.h.q.GetOpenSessionForElder(c.ctx, c.tablet.ElderID)
	switch {
	case err == nil:
		active = map[string]any{"sessionId": sess.ID, "mode": sess.Mode}
	case !errors.Is(err, pgx.ErrNoRows):
		return fmt.Errorf("read open session: %w", err)
	}
	c.send("connection.ready", map[string]any{
		"deviceId": c.tablet.DeviceID, "elderId": c.tablet.ElderID,
		"activeSession": active, "openerVersion": m.Version,
	})
	if active != nil {
		go c.greet(sess.ID)
	}
	return nil
}

// started tells the tablet about a new session and greets the elder.
func (c *conn) started(s db.ConversationSession, etaMinutes *int) {
	c.send("session.started", map[string]any{
		"sessionId": s.ID, "mode": s.Mode, "startedBy": s.StartedBy, "caregiverEtaMinutes": etaMinutes,
	})
	go c.greet(s.ID)
}

func (c *conn) greet(sessionID uuid.UUID) {
	if _, err := c.h.engine.Greet(c.ctx, sessionID, &wsSink{c: c}); err != nil &&
		!errors.Is(err, ErrSessionEnded) && !errors.Is(err, ErrSessionNotFound) {
		c.log().Error("greeting failed", "session_id", sessionID, "err", err)
	}
}

func (c *conn) handle(data []byte) {
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		c.sendError("INVALID_MESSAGE", "JSON이 아닙니다.", "")
		return
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err == nil {
		err = c.h.schema.Validate(inst)
	}
	if err != nil || !clientEvents[env.Type] {
		c.sendError("INVALID_MESSAGE", "메시지 형식이 맞지 않습니다: "+env.Type, env.ID)
		return
	}

	switch env.Type {
	case "ping":
		var d struct{ Nonce string }
		_ = json.Unmarshal(env.Data, &d)
		c.send("pong", map[string]any{"nonce": d.Nonce})
	case "pong":
	case "session.request":
		c.requestCompanion(env.ID)
	case "session.end":
		var d struct{ SessionID uuid.UUID }
		_ = json.Unmarshal(env.Data, &d)
		c.endSession(d.SessionID, env.ID)
	case "elder.text":
		var d struct {
			SessionID uuid.UUID
			ClientID  string
			Text      string
		}
		_ = json.Unmarshal(env.Data, &d)
		go c.elderText(d.SessionID, d.ClientID, d.Text, env.ID)
	case "elder.audio":
		c.elderAudio(env)
	case "client.metrics":
		c.clientMetrics(env.Data)
	case "elder.barge_in":
		var d struct {
			SessionID uuid.UUID
			TurnID    int32
		}
		_ = json.Unmarshal(env.Data, &d)
		if _, err := c.ownSession(d.SessionID); err == nil {
			c.h.engine.BargeIn(d.SessionID, d.TurnID)
		}
	}
}

// clientEvents are the events a tablet may send.
var clientEvents = map[string]bool{
	"ping": true, "pong": true, "session.request": true, "session.end": true,
	"elder.audio": true, "elder.text": true, "elder.barge_in": true, "client.metrics": true,
}

var errNotYours = errors.New("session belongs to another elder")

// ownSession reads a session of this tablet's elder.
func (c *conn) ownSession(id uuid.UUID) (db.ConversationSession, error) {
	s, err := c.h.q.GetSession(c.ctx, id)
	if err != nil {
		return s, err
	}
	if s.ElderID != c.tablet.ElderID {
		return s, errNotYours
	}
	return s, nil
}

func (c *conn) elderText(sessionID uuid.UUID, clientID, text, msgID string) {
	if !c.checkSession(sessionID, msgID) {
		return
	}
	c.turn(sessionID, clientID, text, msgID)
}

// checkSession answers NO_ACTIVE_SESSION unless the session is this elder's.
func (c *conn) checkSession(sessionID uuid.UUID, msgID string) bool {
	_, err := c.ownSession(sessionID)
	switch {
	case err == nil:
		return true
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, errNotYours):
		c.sendError("NO_ACTIVE_SESSION", "진행 중인 세션이 아닙니다.", msgID)
	default:
		c.internal("read session", err, msgID)
	}
	return false
}

func (c *conn) turn(sessionID uuid.UUID, clientID, text, msgID string) {
	_, err := c.h.engine.HandleElderText(c.ctx, sessionID, text, &wsSink{c: c, clientID: clientID})
	switch {
	case err == nil:
	case errors.Is(err, ErrEmptyText):
		c.sendError("INVALID_MESSAGE", "발화가 비어 있습니다.", msgID)
	case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrSessionEnded):
		c.sendError("NO_ACTIVE_SESSION", "진행 중인 세션이 아닙니다.", msgID)
	default:
		c.internal("elder turn", err, msgID)
	}
}

// sttTimeout bounds one recognition; an utterance is at most a minute of audio.
const sttTimeout = 10 * time.Second

// elderAudio takes the binary frame that must follow elder.audio, then runs
// speech recognition and the turn off the read loop.
func (c *conn) elderAudio(env envelope) {
	var d struct {
		SessionID uuid.UUID
		ClientID  string
		Audio     struct {
			Format       string
			SampleRateHz int
			Bytes        int
		}
	}
	_ = json.Unmarshal(env.Data, &d)
	rctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	typ, audio, err := c.ws.Read(rctx)
	cancel()
	if err != nil {
		c.close(websocket.StatusPolicyViolation, "audio frame missing")
		return
	}
	if typ != websocket.MessageBinary || len(audio) != d.Audio.Bytes {
		c.sendError("AUDIO_FRAME_MISSING", "elder.audio 다음에 그 길이의 바이너리 프레임이 와야 합니다.", env.ID)
		if typ == websocket.MessageText {
			c.handle(audio)
		}
		return
	}
	go func() {
		if !c.checkSession(d.SessionID, env.ID) {
			return
		}
		sctx, cancel := context.WithTimeout(c.ctx, sttTimeout)
		start := time.Now()
		text, err := c.h.cfg.STT.Recognize(sctx, audio, d.Audio.Format, d.Audio.SampleRateHz)
		cancel()
		if err != nil {
			c.log().Error("speech recognition failed", "session_id", d.SessionID, "err", err)
			c.sendError("STT_FAILED", "음성 인식에 실패했습니다.", env.ID)
			return
		}
		c.log().Info("speech recognized", "session_id", d.SessionID, "bytes", len(audio), "stt_ms", time.Since(start).Milliseconds())
		if strings.TrimSpace(text) == "" {
			c.sendError("STT_FAILED", "말씀을 알아듣지 못했습니다.", env.ID)
			return
		}
		c.turn(d.SessionID, d.ClientID, text, env.ID)
	}()
}

// clientMetrics logs the latencies the tablet measured for one turn
// (development-process.md stage 4: utterance end to sentence 0, sentence 0
// end to sentence 1).
func (c *conn) clientMetrics(data json.RawMessage) {
	var d struct {
		SessionID                uuid.UUID
		TurnID                   int32
		UtteranceEndToOpenerMs   *int
		OpenerEndToFirstReplyMs  *int
		UtteranceEndToFirstReply *int `json:"utteranceEndToFirstReplyMs"`
	}
	_ = json.Unmarshal(data, &d)
	c.log().Info("turn latency",
		"session_id", d.SessionID, "turn", d.TurnID,
		"utterance_end_to_opener_ms", d.UtteranceEndToOpenerMs,
		"opener_end_to_first_reply_ms", d.OpenerEndToFirstReplyMs,
		"utterance_end_to_first_reply_ms", d.UtteranceEndToFirstReply)
}

// requestCompanion starts a companion session at the elder's button press.
// Bedtime and the daily token limit are checked from stage 6.
func (c *conn) requestCompanion(msgID string) {
	reject := func(reason string) { c.send("session.rejected", map[string]any{"reason": reason}) }

	if _, err := c.h.q.GetOpenSessionForElder(c.ctx, c.tablet.ElderID); err == nil {
		reject("ALREADY_ACTIVE")
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		c.internal("read open session", err, msgID)
		return
	}
	sched, err := c.h.q.GetCompanionSchedule(c.ctx, c.tablet.ElderID)
	switch {
	case err == nil && !sched.Enabled:
		reject("COMPANION_DISABLED")
		return
	case err != nil && !errors.Is(err, pgx.ErrNoRows):
		c.internal("read companion schedule", err, msgID)
		return
	}
	s, err := c.h.q.CreateCompanionSession(c.ctx, db.CreateCompanionSessionParams{
		ElderID: c.tablet.ElderID, StartedBy: "ELDER", PromptVersion: &c.h.cfg.PromptVersion,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		reject("ALREADY_ACTIVE") // a pickup session started at the same moment
		return
	}
	if err != nil {
		c.internal("create companion session", err, msgID)
		return
	}
	c.log().Info("companion session started", "session_id", s.ID)
	c.started(s, nil)
}

// endSession ends a companion session at the elder's stop button. A pickup
// session ends when the caregiver arrives.
func (c *conn) endSession(id uuid.UUID, msgID string) {
	s, err := c.ownSession(id)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, errNotYours) || (err == nil && s.EndedAt != nil) {
		c.sendError("NO_ACTIVE_SESSION", "진행 중인 세션이 아닙니다.", msgID)
		return
	}
	if err != nil {
		c.internal("read session", err, msgID)
		return
	}
	if s.Mode != "COMPANION" {
		c.sendError("INVALID_MESSAGE", "픽업 대기 세션은 선생님이 도착하면 끝납니다.", msgID)
		return
	}
	ended, err := c.h.q.EndSession(c.ctx, db.EndSessionParams{ID: id, EndedReason: ptr("ELDER_DECLINED")})
	if errors.Is(err, pgx.ErrNoRows) {
		c.sendError("NO_ACTIVE_SESSION", "진행 중인 세션이 아닙니다.", msgID)
		return
	}
	if err != nil {
		c.internal("end session", err, msgID)
		return
	}
	c.h.SessionEnded(c.ctx, ended)
}

func (c *conn) internal(what string, err error, msgID string) {
	c.log().Error(what, "err", err)
	c.sendError("INTERNAL", "서버 오류가 발생했습니다.", msgID)
}

func ptr[T any](v T) *T { return &v }

// wsSink sends a turn's events to the tablet, in order, through the
// connection's event goroutine.
type wsSink struct {
	c        *conn
	clientID string
}

func (s *wsSink) Transcript(u db.Utterance) {
	d := map[string]any{"sessionId": u.SessionID, "seq": u.Seq, "utteranceId": u.ID, "text": u.Text}
	if s.clientID != "" {
		d["clientId"] = s.clientID
	}
	s.c.later(func() { s.c.send("elder.transcript", d) })
}

func (s *wsSink) Opener(u db.Utterance, clip db.OpenerClip) {
	s.c.later(func() {
		s.c.send("ai.opener", map[string]any{"sessionId": u.SessionID, "turnId": u.Seq, "clipId": clip.ID, "category": clip.Category})
	})
}

func (s *wsSink) Filler(sessionID uuid.UUID, turn int32, clip db.OpenerClip) {
	s.c.later(func() {
		s.c.send("ai.filler", map[string]any{"sessionId": sessionID, "turnId": turn, "clipId": clip.ID})
	})
}

// ttsTimeout bounds one sentence's synthesis; past it the sentence goes as text.
const ttsTimeout = 5 * time.Second

// Reply synthesizes the sentence and sends it with its MP3. Without audio
// (no Clova key, or TTS failed) the tablet still shows the text.
func (s *wsSink) Reply(u db.Utterance) {
	c := s.c
	c.later(func() {
		d := map[string]any{
			"sessionId": u.SessionID, "turnId": u.Seq, "index": u.ChunkIndex, "utteranceId": u.ID, "text": u.Text, "audio": nil,
		}
		ctx, cancel := context.WithTimeout(c.ctx, ttsTimeout)
		defer cancel()
		mp3, err := c.h.cfg.TTS.Synthesize(ctx, u.Text, c.voice)
		if err != nil {
			if !errors.Is(err, speech.ErrUnavailable) {
				c.log().Error("speech synthesis failed", "utterance_id", u.ID, "err", err)
			}
			c.send("ai.reply", d)
			return
		}
		d["audio"] = map[string]any{"format": "mp3", "bytes": len(mp3)}
		c.sendWithAudio("ai.reply", d, mp3)
	})
}

func (s *wsSink) TurnEnd(sessionID uuid.UUID, turn int32, o Outcome, sentences int) {
	s.c.later(func() {
		s.c.send("ai.turn_end", map[string]any{"sessionId": sessionID, "turnId": turn, "outcome": o, "sentences": sentences})
	})
}
