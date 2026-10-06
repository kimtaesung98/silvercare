package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// maxTokens bounds one reply: one to three short spoken sentences.
const maxTokens = 400

// Anthropic is the Client backed by the Claude API.
type Anthropic struct {
	client anthropic.Client
	model  string
}

// NewAnthropic returns a Client for model.
func NewAnthropic(apiKey, model string, opts ...option.RequestOption) *Anthropic {
	opts = append([]option.RequestOption{option.WithAPIKey(apiKey)}, opts...)
	return &Anthropic{client: anthropic.NewClient(opts...), model: model}
}

var flagConcernTool = anthropic.ToolParam{
	Name: "flag_concern",
	Description: anthropic.String("어르신 말에서 걱정스러운 신호(낙상, 통증, 자해·타해, 길을 잃음, 평소와 다른 심한 혼란이나 불안)를 보면 담당자에게 알립니다. " +
		"어르신께 하는 답과 함께 쓰고, 확실하지 않아도 써도 됩니다."),
	InputSchema: anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"type": map[string]any{
				"type":        "string",
				"enum":        []string{"FALL_MENTION", "PAIN_COMPLAINT", "SELF_OR_OTHER_HARM", "OTHER_ANOMALY"},
				"description": "신호의 종류. 앞의 셋에 맞지 않으면 OTHER_ANOMALY",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "담당자가 읽을 짧은 설명 (어르신이 한 말을 근거로)",
			},
		},
		Required: []string{"type", "reason"},
	},
	EagerInputStreaming: anthropic.Bool(true),
}

func (a *Anthropic) params(req Request) anthropic.MessageNewParams {
	system := make([]anthropic.TextBlockParam, len(req.System))
	for i, s := range req.System {
		system[i] = anthropic.TextBlockParam{Text: s}
	}
	if len(system) > 0 {
		// The mode block is shared by every session; caching it (with the tool
		// definition before it) saves input tokens on every turn.
		system[0].CacheControl = anthropic.NewCacheControlEphemeralParam()
	}

	msgs := make([]anthropic.MessageParam, len(req.Messages))
	for i, m := range req.Messages {
		blocks := []anthropic.ContentBlockParamUnion{anthropic.NewTextBlock(m.Text)}
		if i == len(req.Messages)-1 && req.Continuation != "" {
			blocks = append(blocks, anthropic.NewTextBlock(req.Continuation))
		}
		if m.Role == Assistant {
			msgs[i] = anthropic.NewAssistantMessage(blocks...)
		} else {
			msgs[i] = anthropic.NewUserMessage(blocks...)
		}
	}

	tool := flagConcernTool
	p := anthropic.MessageNewParams{
		Model:      anthropic.Model(a.model),
		MaxTokens:  maxTokens,
		System:     system,
		Messages:   msgs,
		Tools:      []anthropic.ToolUnionParam{{OfTool: &tool}},
		ToolChoice: anthropic.ToolChoiceUnionParam{OfAuto: &anthropic.ToolChoiceAutoParam{}},
		// Automatic caching of the conversation so far: each turn re-reads the
		// previous turns from the cache.
		CacheControl: anthropic.NewCacheControlEphemeralParam(),
	}
	if strings.HasPrefix(a.model, "claude-sonnet-5-5") {
		// Spoken replies need the first sentence fast. "between_tools" is the
		// lowest thinking setting on this model ("disabled" is rejected).
		p.OutputConfig = anthropic.OutputConfigParam{Effort: anthropic.OutputConfigEffortLow}
		p.SetExtraFields(map[string]any{"thinking": map[string]any{"type": "between_tools"}})
	}
	return p
}

// Stream implements Client.
func (a *Anthropic) Stream(ctx context.Context, req Request, onText func(string)) (Result, error) {
	stream := a.client.Messages.NewStreaming(ctx, a.params(req))
	defer stream.Close()

	var msg anthropic.Message
	for stream.Next() {
		ev := stream.Current()
		if err := msg.Accumulate(ev); err != nil {
			return Result{}, fmt.Errorf("accumulate stream: %w", err)
		}
		if d, ok := ev.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
			if t, ok := d.Delta.AsAny().(anthropic.TextDelta); ok && t.Text != "" {
				onText(t.Text)
			}
		}
	}
	if err := stream.Err(); err != nil {
		return Result{}, fmt.Errorf("claude stream: %w", err)
	}

	res := Result{
		Model:        string(msg.Model),
		InputTokens:  int(msg.Usage.InputTokens + msg.Usage.CacheReadInputTokens + msg.Usage.CacheCreationInputTokens),
		OutputTokens: int(msg.Usage.OutputTokens),
		Refused:      msg.StopReason == anthropic.StopReasonRefusal,
	}
	if res.Refused {
		return res, nil
	}
	for _, block := range msg.Content {
		tu, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok || tu.Name != flagConcernTool.Name {
			continue
		}
		c, err := parseConcern(tu.Input)
		if err != nil {
			// The streamed tool input may be cut short; a broken flag is
			// still a flag, so keep it with what can be read.
			c = Concern{Type: "OTHER_ANOMALY", Reason: "flag_concern 입력을 읽지 못함: " + string(tu.Input)}
		}
		res.Concerns = append(res.Concerns, c)
	}
	return res, nil
}

func parseConcern(raw json.RawMessage) (Concern, error) {
	var in struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return Concern{}, err
	}
	if in.Type == "" {
		return Concern{}, errors.New("missing type")
	}
	return Concern{Type: in.Type, Reason: in.Reason}, nil
}

// Unavailable is the Client used when no API key is configured: every call
// fails, so turns end with the fallback sentence.
type Unavailable struct{}

// ErrUnavailable is what Unavailable returns.
var ErrUnavailable = errors.New("claude is not configured (ANTHROPIC_API_KEY is empty)")

// Stream implements Client.
func (Unavailable) Stream(context.Context, Request, func(string)) (Result, error) {
	return Result{}, ErrUnavailable
}
