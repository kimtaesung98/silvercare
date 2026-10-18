package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kimtaesung98/silvercare/v2/chat"
	"github.com/kimtaesung98/silvercare/v2/companion"
)

func post(t *testing.T, url string, body, out any) int {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		_ = json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

// 어르신이 방을 만들고, 조무사가 같은 방에 들어오고, 어르신 말에 말동무가 답하는 것을
// 조무사 쪽 이벤트 흐름으로 확인합니다.
func TestRoomFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &server{hub: chat.NewHub(), newBrain: func() companion.Brain { return &companion.SimpleBrain{} }, ctx: ctx}
	ts := httptest.NewServer(s.routes())
	defer ts.Close()

	var elder, carer joinResponse
	if c := post(t, ts.URL+"/api/rooms", joinRequest{Name: "어르신"}, &elder); c != 200 {
		t.Fatalf("create %d", c)
	}
	if c := post(t, ts.URL+"/api/rooms/"+elder.RoomID+"/join", joinRequest{Name: "조무사"}, &carer); c != 200 {
		t.Fatalf("join %d", c)
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/rooms/"+elder.RoomID+"/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if c := post(t, ts.URL+"/api/rooms/"+elder.RoomID+"/messages", postRequest{From: elder.Me.ID, Text: "오늘 날씨 좋네"}, nil); c != 200 {
		t.Fatalf("post %d", c)
	}

	lines := make(chan chat.Message)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if d, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
				var m chat.Message
				_ = json.Unmarshal([]byte(d), &m)
				lines <- m
			}
		}
	}()
	var got []chat.Message
	for len(got) < 2 {
		select {
		case m := <-lines:
			got = append(got, m)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout, got %+v", got)
		}
	}
	if got[0].From.ID != elder.Me.ID || got[1].From.Kind != chat.AI {
		t.Fatalf("%+v", got)
	}

	if c := post(t, ts.URL+"/api/rooms/"+elder.RoomID+"/messages", postRequest{From: "stranger", Text: "hi"}, nil); c != http.StatusForbidden {
		t.Fatalf("stranger %d", c)
	}
	if c := post(t, ts.URL+"/api/rooms/nope/join", joinRequest{}, nil); c != http.StatusNotFound {
		t.Fatalf("no room %d", c)
	}
}
