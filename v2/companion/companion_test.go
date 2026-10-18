package companion

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kimtaesung98/silvercare/v2/chat"
)

type brainFunc func([]chat.Message) (string, error)

func (f brainFunc) Reply(_ context.Context, _ chat.Participant, h []chat.Message) (string, error) {
	return f(h)
}

func setup(t *testing.T, b Brain) (*chat.Hub, string, *chat.Subscription) {
	t.Helper()
	h := chat.NewHub()
	room := h.CreateRoom()
	_ = h.Join(room, chat.Participant{ID: "e", Name: "어르신", Kind: chat.Human})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	c := &Companion{Hub: h, Self: chat.Participant{ID: "ai", Name: "말동무", Kind: chat.AI}, Brain: b}
	if err := c.Join(ctx, room); err != nil {
		t.Fatal(err)
	}
	sub, _ := h.Subscribe(room)
	t.Cleanup(sub.Close)
	return h, room, sub
}

func next(t *testing.T, sub *chat.Subscription) chat.Message {
	t.Helper()
	select {
	case m := <-sub.C():
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("no message")
		return chat.Message{}
	}
}

func TestCompanionAnswersHumansOnly(t *testing.T) {
	calls := 0
	h, room, sub := setup(t, brainFunc(func(hist []chat.Message) (string, error) {
		calls++
		return "그러셨어요?", nil
	}))
	_, _ = h.Post(room, "e", "오늘 시장에 갔어")
	if m := next(t, sub); m.From.ID != "e" {
		t.Fatalf("first %+v", m)
	}
	m := next(t, sub)
	if m.From.Kind != chat.AI || m.Text != "그러셨어요?" {
		t.Fatalf("reply %+v", m)
	}
	// 자기 말에는 다시 답하지 않음
	select {
	case m := <-sub.C():
		t.Fatalf("unexpected %+v", m)
	case <-time.After(100 * time.Millisecond):
	}
	if calls != 1 {
		t.Fatalf("calls %d", calls)
	}
}

func TestCompanionFallsBackWhenBrainFails(t *testing.T) {
	h, room, sub := setup(t, brainFunc(func([]chat.Message) (string, error) {
		return "", errors.New("down")
	}))
	_, _ = h.Post(room, "e", "여보세요")
	next(t, sub)
	if m := next(t, sub); m.Text != FallbackText {
		t.Fatalf("got %q", m.Text)
	}
}

func TestClaudeBrainRequest(t *testing.T) {
	var got struct {
		Model    string `json:"model"`
		System   string `json:"system"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "k" || r.URL.Path != "/v1/messages" {
			t.Errorf("bad request %s %v", r.URL.Path, r.Header)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":" 네, 반가워요. "}]}`))
	}))
	defer srv.Close()

	self := chat.Participant{ID: "ai", Name: "말동무", Kind: chat.AI}
	elder := chat.Participant{ID: "e", Name: "어르신", Kind: chat.Human}
	carer := chat.Participant{ID: "c", Name: "조무사", Kind: chat.Human}
	b := &ClaudeBrain{APIKey: "k", Model: "m", BaseURL: srv.URL}
	text, err := b.Reply(context.Background(), self, []chat.Message{
		{From: elder, Text: "안녕"}, {From: self, Text: "안녕하세요"}, {From: elder, Text: "배고파"}, {From: carer, Text: "곧 가요"},
	})
	if err != nil || text != "네, 반가워요." {
		t.Fatalf("%q %v", text, err)
	}
	if got.Model != "m" || got.System == "" || len(got.Messages) != 3 {
		t.Fatalf("%+v", got)
	}
	if got.Messages[2].Role != "user" || got.Messages[2].Content != "어르신: 배고파\n조무사: 곧 가요" {
		t.Fatalf("merged %+v", got.Messages[2])
	}
}
