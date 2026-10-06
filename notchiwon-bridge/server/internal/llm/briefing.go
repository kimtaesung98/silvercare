package llm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	"github.com/anthropics/anthropic-sdk-go"
)

//go:embed summary/*.tmpl
var summaryFiles embed.FS

var summaryTemplates = template.Must(template.New("").ParseFS(summaryFiles, "summary/*.tmpl"))

// BriefingPromptVersion identifies the briefing prompt, like PromptVersion
// does for the conversation.
var BriefingPromptVersion = func() string {
	h := sha256.New()
	_ = fs.WalkDir(summaryFiles, "summary", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := summaryFiles.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(b)
		return nil
	})
	return "brief-" + hex.EncodeToString(h.Sum(nil))[:10]
}()

// Line is one utterance of a finished conversation.
type Line struct {
	Elder bool
	Text  string
}

// BriefingInput is a finished pickup session.
type BriefingInput struct {
	ElderName string
	Lines     []Line
	// Escalations describe what the safety filter flagged ("통증 호소: 아파").
	Escalations []string
}

// Keyword is an interest the elder brought up.
type Keyword struct {
	Keyword     string `json:"keyword"`
	Category    string `json:"category"`
	EmotionTone string `json:"emotionTone"`
	Mentions    int    `json:"mentions"`
}

// Briefing is what the caregiver reads on arrival.
type Briefing struct {
	Summary     string    `json:"summary"`
	Keywords    []Keyword `json:"keywords"`
	EmotionTag  string    `json:"emotionTag"`
	EmotionFlag *string   `json:"emotionFlag"`
	// Model is the model that wrote it.
	Model string `json:"-"`
}

// Summarizer writes arrival briefings.
type Summarizer interface {
	Brief(ctx context.Context, in BriefingInput) (Briefing, error)
}

// briefingSchema is the structured output Claude must follow.
var briefingSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"summary", "keywords", "emotionTag", "emotionFlag"},
	"properties": map[string]any{
		"summary": map[string]any{"type": "string"},
		"keywords": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"keyword", "category", "emotionTone", "mentions"},
				"properties": map[string]any{
					"keyword":     map[string]any{"type": "string"},
					"category":    map[string]any{"type": "string", "enum": []string{"FAMILY", "HOBBY", "MEMORY", "FOOD", "HEALTH_OTHER"}},
					"emotionTone": map[string]any{"type": "string", "enum": []string{"POSITIVE", "NEUTRAL", "AVOIDANT"}},
					"mentions":    map[string]any{"type": "integer"},
				},
			},
		},
		"emotionTag":  map[string]any{"type": "string", "enum": []string{"STABLE", "SLIGHTLY_ANXIOUS", "UNUSUAL"}},
		"emotionFlag": map[string]any{"type": []string{"string", "null"}},
	},
}

// maxKeywords caps the keywords kept from one session.
const maxKeywords = 5

// briefingMaxTokens bounds the JSON answer (three sentences and five keywords).
const briefingMaxTokens = 1024

// AnthropicSummarizer is the Summarizer backed by the Claude API.
type AnthropicSummarizer struct {
	*Anthropic
}

// NewAnthropicSummarizer returns a Summarizer for model (Haiku).
func NewAnthropicSummarizer(a *Anthropic) *AnthropicSummarizer { return &AnthropicSummarizer{a} }

func renderSummary(name string, data any) string {
	var b bytes.Buffer
	if err := summaryTemplates.ExecuteTemplate(&b, name, data); err != nil {
		panic(err) // embedded templates, fixed data: a bug
	}
	return strings.TrimSpace(b.String())
}

// briefingParams builds the request; split out for tests.
func (s *AnthropicSummarizer) briefingParams(in BriefingInput) anthropic.MessageNewParams {
	lines := make([]string, len(in.Lines))
	for i, l := range in.Lines {
		who := "AI"
		if l.Elder {
			who = "어르신"
		}
		lines[i] = who + ": " + l.Text
	}
	transcript := renderSummary("transcript.tmpl", struct {
		ElderName   string
		Escalations []string
		Lines       []string
	}{in.ElderName, in.Escalations, lines})
	return anthropic.MessageNewParams{
		Model:     anthropic.Model(s.model),
		MaxTokens: briefingMaxTokens,
		System:    []anthropic.TextBlockParam{{Text: renderSummary("briefing.tmpl", nil)}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(transcript))},
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{Schema: briefingSchema},
		},
	}
}

// Brief implements Summarizer.
func (s *AnthropicSummarizer) Brief(ctx context.Context, in BriefingInput) (Briefing, error) {
	msg, err := s.client.Messages.New(ctx, s.briefingParams(in))
	if err != nil {
		return Briefing{}, fmt.Errorf("claude briefing: %w", err)
	}
	switch msg.StopReason {
	case anthropic.StopReasonRefusal:
		return Briefing{}, fmt.Errorf("claude refused the briefing")
	case anthropic.StopReasonMaxTokens:
		return Briefing{}, fmt.Errorf("briefing cut at max_tokens")
	}
	var text strings.Builder
	for _, block := range msg.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	return parseBriefing(text.String(), string(msg.Model))
}

func parseBriefing(raw, model string) (Briefing, error) {
	var b Briefing
	if err := json.Unmarshal([]byte(raw), &b); err != nil {
		return Briefing{}, fmt.Errorf("decode briefing: %w", err)
	}
	if strings.TrimSpace(b.Summary) == "" {
		return Briefing{}, fmt.Errorf("briefing has no summary")
	}
	kept := b.Keywords[:0]
	for _, k := range b.Keywords {
		k.Keyword = strings.TrimSpace(k.Keyword)
		if k.Keyword == "" {
			continue
		}
		if k.Mentions < 1 {
			k.Mentions = 1
		}
		kept = append(kept, k)
		if len(kept) == maxKeywords {
			break
		}
	}
	b.Keywords = kept
	if b.EmotionFlag != nil && strings.TrimSpace(*b.EmotionFlag) == "" {
		b.EmotionFlag = nil
	}
	b.Model = model
	return b, nil
}

// Brief implements Summarizer: without an API key there is no summary.
func (Unavailable) Brief(context.Context, BriefingInput) (Briefing, error) {
	return Briefing{}, ErrUnavailable
}
