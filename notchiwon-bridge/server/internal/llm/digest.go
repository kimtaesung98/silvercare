package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// DigestInput is one elder's day of companion conversations.
type DigestInput struct {
	ElderName string
	// Date is the local day, as the guardian reads it ("2026-10-06").
	Date         string
	SessionCount int
	// Summaries are the per-session summaries already written, if any.
	Summaries []string
	// Lines are the day's utterances when no summary was written.
	Lines       []Line
	Escalations []string
}

// Digest is the guardian's summary of a day.
type Digest struct {
	Summary     string  `json:"summary"`
	EmotionFlag *string `json:"emotionFlag"`
	Model       string  `json:"-"`
}

// Digester writes guardians' daily digests.
type Digester interface {
	Digest(ctx context.Context, in DigestInput) (Digest, error)
}

// digestMaxTokens bounds the JSON answer (three or four sentences).
const digestMaxTokens = 512

var digestSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"summary", "emotionFlag"},
	"properties": map[string]any{
		"summary":     map[string]any{"type": "string"},
		"emotionFlag": map[string]any{"type": []string{"string", "null"}},
	},
}

// Digest implements Digester.
func (s *AnthropicSummarizer) Digest(ctx context.Context, in DigestInput) (Digest, error) {
	lines := make([]string, len(in.Lines))
	for i, l := range in.Lines {
		who := "AI"
		if l.Elder {
			who = "어르신"
		}
		lines[i] = who + ": " + l.Text
	}
	body := renderSummary("digest_input.tmpl", struct {
		ElderName    string
		Date         string
		SessionCount int
		Summaries    []string
		Lines        []string
		Escalations  []string
	}{in.ElderName, in.Date, in.SessionCount, in.Summaries, lines, in.Escalations})
	msg, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(s.model),
		MaxTokens: digestMaxTokens,
		System:    []anthropic.TextBlockParam{{Text: renderSummary("digest.tmpl", nil)}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(body))},
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{Schema: digestSchema},
		},
	})
	if err != nil {
		return Digest{}, fmt.Errorf("claude daily digest: %w", err)
	}
	switch msg.StopReason {
	case anthropic.StopReasonRefusal:
		return Digest{}, fmt.Errorf("claude refused the daily digest")
	case anthropic.StopReasonMaxTokens:
		return Digest{}, fmt.Errorf("daily digest cut at max_tokens")
	}
	var text strings.Builder
	for _, block := range msg.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	var d Digest
	if err := json.Unmarshal([]byte(text.String()), &d); err != nil {
		return Digest{}, fmt.Errorf("decode daily digest: %w", err)
	}
	if strings.TrimSpace(d.Summary) == "" {
		return Digest{}, fmt.Errorf("daily digest has no summary")
	}
	if d.EmotionFlag != nil && strings.TrimSpace(*d.EmotionFlag) == "" {
		d.EmotionFlag = nil
	}
	d.Model = string(msg.Model)
	return d, nil
}

// Digest implements Digester: without an API key there is no digest.
func (Unavailable) Digest(context.Context, DigestInput) (Digest, error) {
	return Digest{}, ErrUnavailable
}
