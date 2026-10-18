// Package chat는 서비스의 바탕입니다. 카카오톡처럼 대화방, 참가자, 메시지,
// 실시간 전달만 하고, 그 방이 픽업 대기인지 말동무인지, 참가자가 사람인지 AI인지에
// 따라 다르게 움직이지 않습니다.
package chat

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Kind는 참가자가 사람인지 AI인지 적어 둡니다. 바탕은 이 값으로 동작을 바꾸지 않고,
// 화면이나 모듈이 표시·판단에 씁니다.
type Kind string

const (
	Human Kind = "human"
	AI    Kind = "ai"
)

// Participant는 방에 들어온 한 명입니다. 사람이든 AI든 같은 모양입니다.
type Participant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind Kind   `json:"kind"`
}

// Message는 방에 오간 메시지 하나입니다. Seq는 방 안에서 1부터 늘어납니다.
type Message struct {
	RoomID string      `json:"roomId"`
	Seq    int         `json:"seq"`
	From   Participant `json:"from"`
	Text   string      `json:"text"`
	At     time.Time   `json:"at"`
}

var (
	ErrNoRoom    = errors.New("chat: 그런 방이 없습니다")
	ErrNotMember = errors.New("chat: 방 참가자가 아닙니다")
	ErrEmpty     = errors.New("chat: 빈 메시지입니다")
)

// Hub는 방들을 들고 있는 메모리 저장소입니다.
type Hub struct {
	mu     sync.Mutex
	rooms  map[string]*room
	nextID atomic.Int64
	now    func() time.Time
}

type room struct {
	id       string
	members  map[string]Participant
	messages []Message
	subs     map[*Subscription]struct{}
}

// NewHub는 빈 Hub를 만듭니다.
func NewHub() *Hub {
	return &Hub{rooms: map[string]*room{}, now: time.Now}
}

// CreateRoom은 새 방을 만들고 id를 돌려줍니다.
func (h *Hub) CreateRoom() string {
	id := fmt.Sprintf("r%d", h.nextID.Add(1))
	h.mu.Lock()
	h.rooms[id] = &room{id: id, members: map[string]Participant{}, subs: map[*Subscription]struct{}{}}
	h.mu.Unlock()
	return id
}

// Join은 참가자를 방에 넣습니다. 같은 id로 다시 들어오면 이름과 종류만 바뀝니다.
func (h *Hub) Join(roomID string, p Participant) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.rooms[roomID]
	if !ok {
		return ErrNoRoom
	}
	r.members[p.ID] = p
	return nil
}

// Members는 방 참가자 목록을 돌려줍니다.
func (h *Hub) Members(roomID string) ([]Participant, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.rooms[roomID]
	if !ok {
		return nil, ErrNoRoom
	}
	out := make([]Participant, 0, len(r.members))
	for _, p := range r.members {
		out = append(out, p)
	}
	return out, nil
}

// Post는 참가자 fromID가 방에 메시지를 보냅니다. 구독자 모두에게 전달됩니다.
func (h *Hub) Post(roomID, fromID, text string) (Message, error) {
	if text == "" {
		return Message{}, ErrEmpty
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.rooms[roomID]
	if !ok {
		return Message{}, ErrNoRoom
	}
	from, ok := r.members[fromID]
	if !ok {
		return Message{}, ErrNotMember
	}
	m := Message{RoomID: roomID, Seq: len(r.messages) + 1, From: from, Text: text, At: h.now()}
	r.messages = append(r.messages, m)
	for s := range r.subs {
		s.deliver(m)
	}
	return m, nil
}

// History는 방의 메시지를 처음부터 돌려줍니다.
func (h *Hub) History(roomID string) ([]Message, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.rooms[roomID]
	if !ok {
		return nil, ErrNoRoom
	}
	return append([]Message(nil), r.messages...), nil
}

// Subscribe는 방에 새로 오는 메시지를 받는 구독을 엽니다. 다 쓰면 Close를 부릅니다.
func (h *Hub) Subscribe(roomID string) (*Subscription, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.rooms[roomID]
	if !ok {
		return nil, ErrNoRoom
	}
	s := &Subscription{c: make(chan Message, 64), hub: h, room: r}
	r.subs[s] = struct{}{}
	return s, nil
}

// Subscription은 한 구독자에게 가는 메시지 통로입니다.
type Subscription struct {
	c      chan Message
	hub    *Hub
	room   *room
	closed bool
}

// C는 메시지가 오는 채널입니다. Close하면 닫힙니다.
func (s *Subscription) C() <-chan Message { return s.c }

// deliver는 Hub 잠금 안에서 불립니다. 구독자가 너무 느려 통로가 차면 그 구독을
// 끊어 다른 참가자가 기다리지 않게 합니다 (다시 연결해 History로 따라잡습니다).
func (s *Subscription) deliver(m Message) {
	select {
	case s.c <- m:
	default:
		s.closeLocked()
	}
}

// Close는 구독을 끝냅니다. 여러 번 불러도 됩니다.
func (s *Subscription) Close() {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	s.closeLocked()
}

func (s *Subscription) closeLocked() {
	if s.closed {
		return
	}
	s.closed = true
	delete(s.room.subs, s)
	close(s.c)
}
