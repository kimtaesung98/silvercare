package chat

import (
	"errors"
	"testing"
)

func TestPostDeliversToEverySubscriber(t *testing.T) {
	h := NewHub()
	room := h.CreateRoom()
	elder := Participant{ID: "e", Name: "어르신", Kind: Human}
	ai := Participant{ID: "a", Name: "말동무", Kind: AI}
	for _, p := range []Participant{elder, ai} {
		if err := h.Join(room, p); err != nil {
			t.Fatal(err)
		}
	}
	s1, _ := h.Subscribe(room)
	s2, _ := h.Subscribe(room)
	defer s1.Close()
	defer s2.Close()

	if _, err := h.Post(room, "e", "안녕하세요"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Post(room, "a", "네, 안녕하세요."); err != nil {
		t.Fatal(err)
	}
	for _, s := range []*Subscription{s1, s2} {
		m1, m2 := <-s.C(), <-s.C()
		if m1.Seq != 1 || m1.From != elder || m2.Seq != 2 || m2.From != ai {
			t.Fatalf("got %+v, %+v", m1, m2)
		}
	}
	hist, _ := h.History(room)
	if len(hist) != 2 {
		t.Fatalf("history %d", len(hist))
	}
}

func TestPostRules(t *testing.T) {
	h := NewHub()
	room := h.CreateRoom()
	if _, err := h.Post("nope", "x", "hi"); !errors.Is(err, ErrNoRoom) {
		t.Fatalf("no room: %v", err)
	}
	if _, err := h.Post(room, "stranger", "hi"); !errors.Is(err, ErrNotMember) {
		t.Fatalf("not member: %v", err)
	}
	_ = h.Join(room, Participant{ID: "e", Kind: Human})
	if _, err := h.Post(room, "e", ""); !errors.Is(err, ErrEmpty) {
		t.Fatalf("empty: %v", err)
	}
}

func TestSlowSubscriberIsDroppedNotBlocking(t *testing.T) {
	h := NewHub()
	room := h.CreateRoom()
	_ = h.Join(room, Participant{ID: "e", Kind: Human})
	slow, _ := h.Subscribe(room)
	for i := 0; i < 100; i++ {
		if _, err := h.Post(room, "e", "말"); err != nil {
			t.Fatal(err)
		}
	}
	n := 0
	for range slow.C() {
		n++
	}
	if n != 64 {
		t.Fatalf("slow subscriber got %d before being dropped", n)
	}
	slow.Close() // 이미 닫혀 있어도 괜찮아야 함
}
