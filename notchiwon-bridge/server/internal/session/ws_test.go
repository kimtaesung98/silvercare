package session

import (
	"bytes"
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

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/llm"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/opener"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/speech"
)

type wsEnv struct {
	*env
	hub    *Hub
	srv    *httptest.Server
	token  string
	speech *fakeSpeech
}

// fakeSpeech is Clova: it hears a fixed sentence and "synthesizes" the text
// itself as audio.
type fakeSpeech struct {
	mu     sync.Mutex
	heard  string
	sttErr error
	ttsErr error
}

func (f *fakeSpeech) set(heard string, sttErr, ttsErr error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heard, f.sttErr, f.ttsErr = heard, sttErr, ttsErr
}

func (f *fakeSpeech) Recognize(_ context.Context, audio []byte, format string, _ int) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sttErr != nil {
		return "", f.sttErr
	}
	if len(audio) == 0 || format == "" {
		return "", errors.New("no audio")
	}
	return f.heard, nil
}

func (f *fakeSpeech) Synthesize(_ context.Context, text, voice string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ttsErr != nil {
		return nil, f.ttsErr
	}
	return []byte(voice + ":" + text), nil
}

func newWSEnv(t *testing.T) *wsEnv {
	t.Helper()
	e := newEnv(t, DefaultConfig)
	ctx := context.Background()
	// The pickup session newEnv made would greet on connect; start each test idle.
	if _, err := e.q.EndSession(ctx, db.EndSessionParams{ID: e.session, EndedReason: ptr("CAREGIVER_ARRIVED")}); err != nil {
		t.Fatal(err)
	}
	token := auth.NewDeviceToken()
	if _, err := e.q.CreateElderTablet(ctx, db.CreateElderTabletParams{ElderID: &e.elderID, TokenHash: auth.HashDeviceToken(token)}); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSpeech{sttErr: speech.ErrUnavailable, ttsErr: speech.ErrUnavailable}
	hub, err := NewHub(e.q, e.engine, opener.NewLibrary(e.q), HubConfig{PromptVersion: llm.PromptVersion, STT: fs, TTS: fs},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(hub)
	t.Cleanup(srv.Close)
	return &wsEnv{env: e, hub: hub, srv: srv, token: token, speech: fs}
}

type client struct {
	t      *testing.T
	ws     *websocket.Conn
	schema *jsonschema.Schema
}

func (w *wsEnv) dial(t *testing.T) *client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(w.srv.URL, "http"), &websocket.DialOptions{
		HTTPHeader: http.Header{"Authorization": {"Bearer " + w.token}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ws.CloseNow() })
	return &client{t: t, ws: ws, schema: w.hub.schema}
}

type event struct {
	Type  string
	Data  map[string]any
	Audio []byte // the binary frame that followed, if any
}

// next reads one server event and checks it against ws-events.schema.json.
func (c *client) next() event {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	typ, data, err := c.ws.Read(ctx)
	if err != nil {
		c.t.Fatalf("read: %v", err)
	}
	if typ != websocket.MessageText {
		c.t.Fatalf("frame type %v", typ)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		c.t.Fatal(err)
	}
	if err := c.schema.Validate(inst); err != nil {
		c.t.Fatalf("server sent an invalid event %s: %v", data, err)
	}
	var ev event
	if err := json.Unmarshal(data, &ev); err != nil {
		c.t.Fatal(err)
	}
	if a, ok := ev.Data["audio"].(map[string]any); ok {
		typ, bin, err := c.ws.Read(ctx)
		if err != nil || typ != websocket.MessageBinary || float64(len(bin)) != a["bytes"] {
			c.t.Fatalf("audio frame after %s: type %v, %d bytes, err %v; want %v bytes", ev.Type, typ, len(bin), err, a["bytes"])
		}
		ev.Audio = bin
	}
	return ev
}

// expect reads events until one of type typ arrives, failing on an error event.
func (c *client) expect(typ string) event {
	c.t.Helper()
	for {
		ev := c.next()
		if ev.Type == typ {
			return ev
		}
		if ev.Type == "error" {
			c.t.Fatalf("error while waiting for %s: %v", typ, ev.Data)
		}
	}
}

func (c *client) send(typ string, data any) {
	c.t.Helper()
	b, _ := json.Marshal(map[string]any{"type": typ, "id": "m-1", "data": data})
	if err := c.ws.Write(context.Background(), websocket.MessageText, b); err != nil {
		c.t.Fatal(err)
	}
}

func TestWSUnauthorized(t *testing.T) {
	w := newWSEnv(t)
	for _, h := range []string{"", "Bearer nope"} {
		req, _ := http.NewRequest(http.MethodGet, w.srv.URL, nil)
		if h != "" {
			req.Header.Set("Authorization", h)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%q: status %d", h, resp.StatusCode)
		}
	}
}

// A companion conversation over the socket, in text mode.
func TestWSCompanionConversation(t *testing.T) {
	w := newWSEnv(t)
	w.llm.chunks = []string{"안녕하세요, 김순자님. 오늘 기분은 어떠세요?"}
	c := w.dial(t)

	ready := c.expect("connection.ready")
	if ready.Data["elderId"] != w.elderID.String() || ready.Data["activeSession"] != nil || ready.Data["openerVersion"] == "" {
		t.Fatalf("ready = %v", ready.Data)
	}

	c.send("session.request", map[string]any{"mode": "COMPANION"})
	started := c.expect("session.started")
	if started.Data["mode"] != "COMPANION" || started.Data["startedBy"] != "ELDER" {
		t.Fatalf("started = %v", started.Data)
	}
	sid := started.Data["sessionId"].(string)
	if r := c.expect("ai.reply"); r.Data["turnId"] != 0.0 || r.Data["index"] != 1.0 || r.Data["audio"] != nil {
		t.Errorf("greeting = %v", r.Data)
	}
	c.expect("ai.reply")
	if end := c.expect("ai.turn_end"); end.Data["outcome"] != "COMPLETED" || end.Data["sentences"] != 2.0 {
		t.Errorf("greeting end = %v", end.Data)
	}

	// A second press while talking is rejected.
	c.send("session.request", map[string]any{"mode": "COMPANION"})
	if rej := c.expect("session.rejected"); rej.Data["reason"] != "ALREADY_ACTIVE" {
		t.Errorf("rejected = %v", rej.Data)
	}

	w.llm.chunks = []string{"손주분이 오셨군요. 무엇을 하셨어요?"}
	c.send("elder.text", map[string]any{"sessionId": sid, "clientId": "u-1", "text": "손주가 어제 왔어"})
	tr := c.expect("elder.transcript")
	if tr.Data["seq"] != 1.0 || tr.Data["clientId"] != "u-1" || tr.Data["text"] != "손주가 어제 왔어" {
		t.Errorf("transcript = %v", tr.Data)
	}
	if op := c.next(); op.Type != "ai.opener" || op.Data["turnId"] != 2.0 || op.Data["category"] != "RECALL" {
		t.Errorf("opener = %v", op)
	}
	for i := 1.0; i <= 2; i++ {
		if r := c.next(); r.Type != "ai.reply" || r.Data["index"] != i {
			t.Errorf("reply %v = %v", i, r)
		}
	}
	if end := c.next(); end.Type != "ai.turn_end" || end.Data["outcome"] != "COMPLETED" {
		t.Errorf("turn end = %v", end)
	}

	// A rule match: the ESCALATION clip, no Claude.
	c.send("elder.text", map[string]any{"sessionId": sid, "clientId": "u-2", "text": "다리가 너무 아파"})
	c.expect("elder.transcript")
	if op := c.next(); op.Type != "ai.opener" || op.Data["category"] != "ESCALATION" {
		t.Errorf("opener = %v", op)
	}
	if end := c.next(); end.Type != "ai.turn_end" || end.Data["outcome"] != "ESCALATED" || end.Data["sentences"] != 0.0 {
		t.Errorf("turn end = %v", end)
	}

	c.send("session.end", map[string]any{"sessionId": sid})
	if ended := c.expect("session.ended"); ended.Data["reason"] != "ELDER_DECLINED" {
		t.Errorf("ended = %v", ended.Data)
	}
	c.send("elder.text", map[string]any{"sessionId": sid, "clientId": "u-3", "text": "어디 갔어"})
	if ev := c.next(); ev.Type != "error" || ev.Data["code"] != "NO_ACTIVE_SESSION" || ev.Data["replyTo"] != "m-1" {
		t.Errorf("after end = %v", ev)
	}
}

func TestWSInvalidMessages(t *testing.T) {
	w := newWSEnv(t)
	c := w.dial(t)
	c.expect("connection.ready")

	for _, raw := range []string{
		`not json`,
		`{"type":"ai.reply","data":{}}`,
		`{"type":"elder.text","id":"x","data":{"sessionId":"nope","clientId":"u","text":"안녕"}}`,
		`{"type":"session.request","data":{"mode":"PICKUP_BRIDGE"}}`,
	} {
		if err := c.ws.Write(context.Background(), websocket.MessageText, []byte(raw)); err != nil {
			t.Fatal(err)
		}
		if ev := c.next(); ev.Type != "error" || ev.Data["code"] != "INVALID_MESSAGE" {
			t.Errorf("%s: got %v", raw, ev)
		}
	}

	// Someone else's session is not this tablet's to talk in.
	other := uuid.New()
	c.send("elder.text", map[string]any{"sessionId": other, "clientId": "u", "text": "안녕"})
	if ev := c.next(); ev.Data["code"] != "NO_ACTIVE_SESSION" {
		t.Errorf("unknown session: %v", ev)
	}

	c.send("ping", map[string]any{"nonce": "n-7"})
	if ev := c.next(); ev.Type != "pong" || ev.Data["nonce"] != "n-7" {
		t.Errorf("pong = %v", ev)
	}

	// Without Clova the audio is refused, but the frame pairing is still enforced.
	sess, err := w.q.CreateCompanionSession(context.Background(), db.CreateCompanionSessionParams{ElderID: w.elderID, StartedBy: "ELDER"})
	if err != nil {
		t.Fatal(err)
	}
	c.send("elder.audio", map[string]any{"sessionId": sess.ID, "clientId": "u", "audio": map[string]any{"format": "pcm16", "bytes": 4}})
	if err := c.ws.Write(context.Background(), websocket.MessageBinary, []byte{1, 2, 3, 4}); err != nil {
		t.Fatal(err)
	}
	if ev := c.expect("error"); ev.Data["code"] != "STT_FAILED" {
		t.Errorf("audio: %v", ev)
	}
	c.send("elder.audio", map[string]any{"sessionId": sess.ID, "clientId": "u", "audio": map[string]any{"format": "pcm16", "bytes": 8}})
	if err := c.ws.Write(context.Background(), websocket.MessageBinary, []byte{1, 2}); err != nil {
		t.Fatal(err)
	}
	if ev := c.expect("error"); ev.Data["code"] != "AUDIO_FRAME_MISSING" {
		t.Errorf("short audio: %v", ev)
	}

	c.send("client.metrics", map[string]any{"sessionId": sess.ID, "turnId": 2, "utteranceEndToOpenerMs": 600, "openerEndToFirstReplyMs": nil})
	c.send("ping", map[string]any{"nonce": "after-metrics"})
	if ev := c.next(); ev.Type != "pong" {
		t.Errorf("client.metrics answered with %v", ev)
	}
}

// A spoken turn: the audio is recognized, and every reply carries its MP3.
func TestWSVoiceTurn(t *testing.T) {
	w := newWSEnv(t)
	w.speech.set("손주가 어제 왔어", nil, nil)
	w.llm.chunks = []string{"손주분이 오셨군요. 무엇을 하셨어요?"}
	c := w.dial(t)
	c.expect("connection.ready")
	sess, err := w.q.CreateCompanionSession(context.Background(), db.CreateCompanionSessionParams{ElderID: w.elderID, StartedBy: "ELDER"})
	if err != nil {
		t.Fatal(err)
	}
	w.say(t, 0, "AI", "안녕하세요.") // already greeted

	pcm := make([]byte, 3200)
	c.send("elder.audio", map[string]any{"sessionId": sess.ID, "clientId": "u-9", "audio": map[string]any{"format": "pcm16", "sampleRateHz": 16000, "bytes": len(pcm)}})
	if err := c.ws.Write(context.Background(), websocket.MessageBinary, pcm); err != nil {
		t.Fatal(err)
	}
	tr := c.expect("elder.transcript")
	if tr.Data["text"] != "손주가 어제 왔어" || tr.Data["clientId"] != "u-9" {
		t.Errorf("transcript = %v", tr.Data)
	}
	c.expect("ai.opener")
	for _, want := range []string{"손주분이 오셨군요.", "무엇을 하셨어요?"} {
		r := c.next()
		if r.Type != "ai.reply" || r.Data["text"] != want || string(r.Audio) != "default:"+want {
			t.Errorf("reply = %v, audio %q", r.Data, r.Audio)
		}
	}
	if end := c.next(); end.Type != "ai.turn_end" || end.Data["outcome"] != "COMPLETED" {
		t.Errorf("end = %v", end)
	}

	// A failing synthesis still delivers the sentence as text.
	w.speech.set("오늘 날씨 좋네", nil, errors.New("clova 500"))
	c.send("elder.audio", map[string]any{"sessionId": sess.ID, "clientId": "u-10", "audio": map[string]any{"format": "pcm16", "bytes": len(pcm)}})
	if err := c.ws.Write(context.Background(), websocket.MessageBinary, pcm); err != nil {
		t.Fatal(err)
	}
	if r := c.expect("ai.reply"); r.Data["audio"] != nil || r.Audio != nil {
		t.Errorf("reply without tts = %v", r.Data)
	}
}

// The visit service's pickup sessions reach the tablet through the hub.
func TestWSPickupNotifications(t *testing.T) {
	w := newWSEnv(t)
	w.llm.chunks = []string{"선생님이 곧 오세요."}
	c := w.dial(t)
	c.expect("connection.ready")
	ctx := context.Background()

	var visitID uuid.UUID
	if err := w.pool.QueryRow(ctx, `SELECT id FROM visit WHERE elder_id = $1`, w.elderID).Scan(&visitID); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx, `INSERT INTO visit (elder_id, caregiver_id, scheduled_time) SELECT elder_id, caregiver_id, now() FROM visit WHERE id = $1`, visitID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx, `SELECT id FROM visit WHERE elder_id = $1 AND id <> $2`, w.elderID, visitID).Scan(&visitID); err != nil {
		t.Fatal(err)
	}
	sess, err := w.q.CreatePickupSession(ctx, db.CreatePickupSessionParams{VisitID: &visitID, ElderID: w.elderID, TriggerEtaMinutes: ptr(int32(12))})
	if err != nil {
		t.Fatal(err)
	}

	w.hub.SessionStarted(ctx, sess, 12)
	if s := c.expect("session.started"); s.Data["mode"] != "PICKUP_BRIDGE" || s.Data["caregiverEtaMinutes"] != 12.0 {
		t.Errorf("started = %v", s.Data)
	}
	c.expect("ai.turn_end") // greeting

	w.hub.EtaUpdated(ctx, sess, 12) // unchanged: not sent
	w.hub.EtaUpdated(ctx, sess, 9)
	if eta := c.next(); eta.Type != "caregiver.eta" || eta.Data["minutes"] != 9.0 || eta.Data["visitId"] != visitID.String() {
		t.Errorf("eta = %v", eta)
	}

	c.send("session.end", map[string]any{"sessionId": sess.ID})
	if ev := c.next(); ev.Type != "error" || ev.Data["code"] != "INVALID_MESSAGE" {
		t.Errorf("ending a pickup session from the tablet: %v", ev)
	}

	ended, err := w.q.EndSession(ctx, db.EndSessionParams{ID: sess.ID, EndedReason: ptr("CAREGIVER_ARRIVED")})
	if err != nil {
		t.Fatal(err)
	}
	w.hub.SessionEnded(ctx, ended)
	if ev := c.next(); ev.Type != "session.ended" || ev.Data["reason"] != "CAREGIVER_ARRIVED" {
		t.Errorf("ended = %v", ev)
	}
}

// Reconnecting replaces the old connection and resumes the open session,
// greeting it if nothing was said yet.
func TestWSReconnectGreetsOpenSession(t *testing.T) {
	w := newWSEnv(t)
	first := w.dial(t)
	first.expect("connection.ready")

	sess, err := w.q.CreateCompanionSession(context.Background(), db.CreateCompanionSessionParams{ElderID: w.elderID, StartedBy: "SCHEDULE"})
	if err != nil {
		t.Fatal(err)
	}
	second := w.dial(t)
	ready := second.expect("connection.ready")
	active, _ := ready.Data["activeSession"].(map[string]any)
	if active["sessionId"] != sess.ID.String() {
		t.Errorf("activeSession = %v", ready.Data["activeSession"])
	}
	if r := second.expect("ai.reply"); r.Data["turnId"] != 0.0 {
		t.Errorf("greeting = %v", r.Data)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		if _, _, err := first.ws.Read(ctx); err != nil {
			if websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
				t.Errorf("old connection closed with %v", err)
			}
			break
		}
	}
}
