// Command elder-cli is a development client for /ws/elder in text mode: type
// what the elder says and read the AI's turn as the tablet would receive it.
//
//	go run ./cmd/elder-cli -token <tablet token from cmd/admin>
//
// Lines starting with / are commands: /start (companion button), /end,
// /barge (talk over the current turn), /quit. Anything else is elder.text.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

func main() {
	server := flag.String("server", "http://localhost:8000", "server base URL")
	token := flag.String("token", os.Getenv("ELDER_TABLET_TOKEN"), "tablet device token (or ELDER_TABLET_TOKEN)")
	flag.Parse()
	if *token == "" {
		fmt.Fprintln(os.Stderr, "need -token (create one with: go run ./cmd/admin create-tablet ...)")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, *server, *token); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type envelope struct {
	Type string          `json:"type"`
	ID   string          `json:"id,omitempty"`
	Data json.RawMessage `json:"data"`
}

type state struct {
	mu      sync.Mutex
	session string
	turn    int
	clips   map[string]string // clip id → text
}

func run(ctx context.Context, server, token string) error {
	auth := http.Header{"Authorization": {"Bearer " + token}}
	st := &state{turn: -1, clips: map[string]string{}}
	if err := loadClips(ctx, server, auth, st); err != nil {
		return err
	}

	u, err := url.Parse(server)
	if err != nil {
		return err
	}
	u.Scheme = map[string]string{"https": "wss"}[u.Scheme]
	if u.Scheme == "" {
		u.Scheme = "ws"
	}
	u.Path = "/ws/elder"
	ws, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{HTTPHeader: auth})
	if err != nil {
		return fmt.Errorf("connect %s: %w", u, err)
	}
	defer ws.CloseNow()

	var seq int
	send := func(typ string, data any) {
		seq++
		b, _ := json.Marshal(map[string]any{"type": typ, "id": fmt.Sprintf("cli-%d", seq), "data": data})
		if err := ws.Write(ctx, websocket.MessageText, b); err != nil {
			fmt.Fprintln(os.Stderr, "send:", err)
		}
	}

	go func() {
		in := bufio.NewScanner(os.Stdin)
		for in.Scan() {
			line := strings.TrimSpace(in.Text())
			st.mu.Lock()
			sid, turn := st.session, st.turn
			st.mu.Unlock()
			switch {
			case line == "":
			case line == "/quit":
				ws.Close(websocket.StatusNormalClosure, "bye")
				return
			case line == "/start":
				send("session.request", map[string]any{"mode": "COMPANION"})
			case sid == "":
				fmt.Println("(진행 중인 세션이 없습니다. /start로 말동무를 시작하세요)")
			case line == "/end":
				send("session.end", map[string]any{"sessionId": sid})
			case line == "/barge":
				send("elder.barge_in", map[string]any{"sessionId": sid, "turnId": max(turn, 0), "playedIndex": 0})
			default:
				send("elder.text", map[string]any{"sessionId": sid, "clientId": fmt.Sprintf("u-%d", time.Now().UnixMilli()), "text": line})
			}
		}
	}()

	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return nil
			}
			return err
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			fmt.Println("?", string(data))
			continue
		}
		var d map[string]any
		_ = json.Unmarshal(env.Data, &d)
		if env.Type == "ping" {
			send("pong", map[string]any{"nonce": d["nonce"]})
			continue
		}
		show(st, env.Type, d)
	}
}

func show(st *state, typ string, d map[string]any) {
	st.mu.Lock()
	defer st.mu.Unlock()
	switch typ {
	case "connection.ready":
		if a, ok := d["activeSession"].(map[string]any); ok {
			st.session = fmt.Sprint(a["sessionId"])
			fmt.Printf("연결됨. 진행 중 세션 %s (%s)\n", st.session, a["mode"])
		} else {
			fmt.Println("연결됨. 진행 중 세션 없음 (/start로 말동무 시작)")
		}
	case "session.started":
		st.session = fmt.Sprint(d["sessionId"])
		fmt.Printf("== 세션 시작 %s (%s, %s)\n", d["mode"], d["startedBy"], st.session)
	case "session.rejected":
		fmt.Printf("== 세션 거절: %s\n", d["reason"])
	case "session.ended":
		st.session = ""
		fmt.Printf("== 세션 종료: %s\n", d["reason"])
	case "elder.transcript":
		fmt.Printf("어르신 [%v] %s\n", d["seq"], d["text"])
	case "ai.opener":
		st.turn = toInt(d["turnId"])
		fmt.Printf("AI    [%v.0] %s  (%s, 캐시 음성)\n", d["turnId"], st.clips[fmt.Sprint(d["clipId"])], d["category"])
	case "ai.filler":
		fmt.Printf("AI    [추임새] %s\n", st.clips[fmt.Sprint(d["clipId"])])
	case "ai.reply":
		st.turn = toInt(d["turnId"])
		fmt.Printf("AI    [%v.%v] %s\n", d["turnId"], d["index"], d["text"])
	case "ai.turn_end":
		fmt.Printf("      (턴 %v 끝: %s, 문장 %v개)\n", d["turnId"], d["outcome"], d["sentences"])
	case "caregiver.eta":
		fmt.Printf("== 선생님 도착까지 %v분\n", d["minutes"])
	case "error":
		fmt.Printf("!! %s: %s\n", d["code"], d["message"])
	default:
		fmt.Printf("%s %v\n", typ, d)
	}
}

func toInt(v any) int {
	f, _ := v.(float64)
	return int(f)
}

// loadClips reads sentence 0 texts so opener events can be shown as text.
func loadClips(ctx context.Context, server string, auth http.Header, st *state) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(server, "/")+"/tablet/opener-clips", nil)
	if err != nil {
		return err
	}
	req.Header = auth.Clone()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("opener clips: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("opener clips: %s", resp.Status)
	}
	var m struct {
		Clips []struct{ ID, Text string }
	}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return err
	}
	for _, c := range m.Clips {
		st.clips[c.ID] = c.Text
	}
	return nil
}
