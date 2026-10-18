package companion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kimtaesung98/silvercare/v2/chat"
)

// SimpleBrain은 API 키 없이도 대화가 돌도록 정해진 맞장구를 돌려 씁니다.
type SimpleBrain struct {
	mu sync.Mutex
	n  int
}

var simpleReplies = []string{
	"그러셨어요? 조금 더 이야기해 주세요.",
	"아이고, 그러셨구나요. 그때 기분이 어떠셨어요?",
	"네, 네. 잘 듣고 있어요.",
	"그럼요. 오늘은 어떻게 지내셨어요?",
}

func (b *SimpleBrain) Reply(context.Context, chat.Participant, []chat.Message) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	r := simpleReplies[b.n%len(simpleReplies)]
	b.n++
	return r, nil
}

// SystemPrompt는 말동무의 역할과 말투입니다.
const SystemPrompt = `당신은 어르신과 대화하는 다정한 말동무입니다.
어르신이 혼자 기다리시는 시간에 곁에서 편안하게 이야기를 나눕니다.

말하는 방식:
- 답은 소리 내어 읽힐 수 있습니다. 짧고 쉬운 문장으로, 존댓말로 말하세요.
- 한 번에 1~3문장만 말하고, 질문은 한 번에 하나만 하세요.
- 목록, 기호, 이모지, 괄호 설명은 쓰지 마세요.
- 의료적 진단이나 조언은 하지 말고, 모르는 사실을 단정하지 마세요.
- 아프다거나 넘어졌다는 말에는 스스로 판단하지 말고 "지금 많이 힘드시죠, 곧 도와드릴 분께 알릴게요"처럼 짧게 안심시키세요.`

// ClaudeBrain은 Anthropic Messages API로 답을 만듭니다.
type ClaudeBrain struct {
	APIKey string
	Model  string
	// BaseURL은 테스트에서 바꿔 끼웁니다. 비우면 https://api.anthropic.com.
	BaseURL string
	Client  *http.Client
}

func (b *ClaudeBrain) Reply(ctx context.Context, self chat.Participant, history []chat.Message) (string, error) {
	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	// 말동무가 쓴 말은 assistant, 나머지 참가자의 말은 user입니다. 같은 역할이
	// 이어지면 한 덩어리로 합칩니다 (API가 역할이 번갈아 오기를 기대함).
	var msgs []msg
	for _, m := range history {
		role, text := "user", m.Text
		if m.From.ID == self.ID {
			role = "assistant"
		} else {
			text = m.From.Name + ": " + m.Text
		}
		if n := len(msgs); n > 0 && msgs[n-1].Role == role {
			msgs[n-1].Content += "\n" + text
			continue
		}
		msgs = append(msgs, msg{role, text})
	}
	if len(msgs) == 0 || msgs[0].Role != "user" {
		msgs = append([]msg{{"user", "(대화를 시작합니다)"}}, msgs...)
	}

	body, err := json.Marshal(map[string]any{
		"model":      b.Model,
		"max_tokens": 300,
		"system":     SystemPrompt,
		"messages":   msgs,
	})
	if err != nil {
		return "", err
	}
	base := b.BaseURL
	if base == "" {
		base = "https://api.anthropic.com"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", b.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := b.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude: %s: %s", resp.Status, raw)
	}
	var out struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, c := range out.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return strings.TrimSpace(sb.String()), nil
}
