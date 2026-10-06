package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// HistoryInput is the part of a conversation to fold into a summary.
type HistoryInput struct {
	ElderName string
	// Previous is the summary written earlier, if any.
	Previous string
	Lines    []Line
}

// historyMaxTokens bounds the summary (three to five sentences).
const historyMaxTokens = 512

// Compacter summarizes the early part of a long conversation.
type Compacter interface {
	Summarize(ctx context.Context, in HistoryInput) (string, error)
}

// Summarize implements Compacter.
func (s *AnthropicSummarizer) Summarize(ctx context.Context, in HistoryInput) (string, error) {
	lines := make([]string, len(in.Lines))
	for i, l := range in.Lines {
		who := "AI"
		if l.Elder {
			who = "어르신"
		}
		lines[i] = who + ": " + l.Text
	}
	body := renderSummary("history_input.tmpl", struct {
		ElderName string
		Previous  string
		Lines     []string
	}{in.ElderName, in.Previous, lines})
	msg, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(s.model),
		MaxTokens: historyMaxTokens,
		System:    []anthropic.TextBlockParam{{Text: renderSummary("history.tmpl", nil)}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(body))},
	})
	if err != nil {
		return "", fmt.Errorf("claude history summary: %w", err)
	}
	if msg.StopReason == anthropic.StopReasonRefusal {
		return "", fmt.Errorf("claude refused the history summary")
	}
	var text strings.Builder
	for _, block := range msg.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	out := strings.TrimSpace(text.String())
	if out == "" {
		return "", fmt.Errorf("history summary is empty")
	}
	return out, nil
}

// Summarize implements Compacter: without an API key there is no summary.
func (Unavailable) Summarize(context.Context, HistoryInput) (string, error) {
	return "", ErrUnavailable
}
