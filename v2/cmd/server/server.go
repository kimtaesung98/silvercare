package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kimtaesung98/silvercare/v2/chat"
	"github.com/kimtaesung98/silvercare/v2/companion"
	"github.com/kimtaesung98/silvercare/v2/safety"
)

//go:embed web
var webFS embed.FS

// server는 채팅 바탕을 HTTP로 열어 줍니다. 서비스 규칙은 하나입니다: 방을 만든
// 사람(어르신) 곁에 말동무 AI와 안전 지킴이를 함께 넣고, 말동무는 어르신에게만 답합니다.
type server struct {
	hub      *chat.Hub
	newBrain func() companion.Brain
	log      *slog.Logger
	// ctx가 끝나면 말동무들도 방에서 나갑니다.
	ctx    context.Context
	nextID atomic.Int64

	// rooms는 이 서비스가 연 방 목록입니다. 조무사 화면이 어르신 방을 찾을 때 씁니다.
	mu    sync.Mutex
	rooms []roomInfo
}

// roomInfo는 열린 방 하나의 안내입니다. 바탕(chat)이 아니라 서비스가 들고 있습니다.
type roomInfo struct {
	ID     string    `json:"id"`
	Elder  string    `json:"elder"`
	Opened time.Time `json:"opened"`
}

func newServer(ctx context.Context, newBrain func() companion.Brain, log *slog.Logger) *server {
	hub := chat.NewHub()
	hub.Use(safety.Module{})
	return &server{hub: hub, newBrain: newBrain, log: log, ctx: ctx}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(webFS, "web")
	mux.Handle("GET /", http.FileServerFS(static))
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/rooms", s.listRooms)
	mux.HandleFunc("POST /api/rooms", s.createRoom)
	mux.HandleFunc("POST /api/rooms/{room}/join", s.joinRoom)
	mux.HandleFunc("POST /api/rooms/{room}/messages", s.postMessage)
	mux.HandleFunc("GET /api/rooms/{room}/events", s.events)
	return mux
}

type joinRequest struct {
	Name string `json:"name"`
}

type joinResponse struct {
	RoomID string           `json:"roomId"`
	Me     chat.Participant `json:"me"`
}

// createRoom은 새 방을 만들고 요청한 사람, 말동무 AI, 안전 지킴이를 넣습니다.
func (s *server) createRoom(w http.ResponseWriter, r *http.Request) {
	var req joinRequest
	if !decode(w, r, &req) {
		return
	}
	roomID := s.hub.CreateRoom()
	me, err := s.join(roomID, req.Name)
	if err != nil {
		httpError(w, err)
		return
	}
	c := &companion.Companion{
		Hub:        s.hub,
		Self:       chat.Participant{ID: "ai-" + roomID, Name: "말동무", Kind: chat.AI},
		Brain:      s.newBrain(),
		Log:        s.log,
		AnswerOnly: me.ID,
	}
	if err := c.Join(s.ctx, roomID); err != nil {
		httpError(w, err)
		return
	}
	if err := (&safety.Guard{Hub: s.hub, Log: s.log}).Join(s.ctx, roomID); err != nil {
		httpError(w, err)
		return
	}
	s.mu.Lock()
	s.rooms = append(s.rooms, roomInfo{ID: roomID, Elder: me.Name, Opened: time.Now()})
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, joinResponse{RoomID: roomID, Me: me})
}

// health는 태블릿 보호자 화면이 "인터넷이 연결됐어요"를 확인할 때 부릅니다.
func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// listRooms는 열린 방을 최근 것부터 돌려줍니다. 조무사가 브라우저에서 어르신 방을 고를 때 씁니다.
func (s *server) listRooms(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	out := make([]roomInfo, 0, len(s.rooms))
	for i := len(s.rooms) - 1; i >= 0; i-- {
		out = append(out, s.rooms[i])
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, out)
}

// joinRoom은 이미 있는 방에 사람 한 명을 더 넣습니다 (예: 조무사, 보호자).
func (s *server) joinRoom(w http.ResponseWriter, r *http.Request) {
	var req joinRequest
	if !decode(w, r, &req) {
		return
	}
	roomID := r.PathValue("room")
	me, err := s.join(roomID, req.Name)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, joinResponse{RoomID: roomID, Me: me})
}

func (s *server) join(roomID, name string) (chat.Participant, error) {
	if name == "" {
		name = "어르신"
	}
	p := chat.Participant{ID: fmt.Sprintf("h%d", s.nextID.Add(1)), Name: name, Kind: chat.Human}
	return p, s.hub.Join(roomID, p)
}

type postRequest struct {
	From string `json:"from"`
	Text string `json:"text"`
}

func (s *server) postMessage(w http.ResponseWriter, r *http.Request) {
	var req postRequest
	if !decode(w, r, &req) {
		return
	}
	m, err := s.hub.Post(r.PathValue("room"), req.From, req.Text)
	if err != nil {
		httpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// events는 Server-Sent Events로 지난 메시지를 먼저, 그다음 새 메시지를 보냅니다.
func (s *server) events(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("room")
	// 구독을 먼저 열고 기록을 읽어야 그 사이에 온 메시지를 놓치지 않습니다.
	sub, err := s.hub.Subscribe(roomID)
	if err != nil {
		httpError(w, err)
		return
	}
	defer sub.Close()
	history, err := s.hub.History(roomID)
	if err != nil {
		httpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, _ := w.(http.Flusher)
	last := 0
	send := func(m chat.Message) {
		if m.Seq <= last {
			return
		}
		last = m.Seq
		b, _ := json.Marshal(m)
		fmt.Fprintf(w, "data: %s\n\n", b)
	}
	for _, m := range history {
		send(m)
	}
	if flusher != nil {
		flusher.Flush()
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case m, ok := <-sub.C():
			if !ok {
				return
			}
			send(m)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "잘못된 요청입니다"})
		return false
	}
	return true
}

func httpError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, chat.ErrNoRoom):
		status = http.StatusNotFound
	case errors.Is(err, chat.ErrNotMember):
		status = http.StatusForbidden
	case errors.Is(err, chat.ErrEmpty):
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
