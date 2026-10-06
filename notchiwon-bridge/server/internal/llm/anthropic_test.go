package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"
)

// sse writes a canned Messages API stream: two text deltas, then a
// flag_concern call whose input arrives in two pieces.
func sse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	events := []string{
		`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-5-5","content":[],"stop_reason":null,"usage":{"input_tokens":120,"cache_read_input_tokens":30,"output_tokens":1}}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"손주분이 오셨군요. "}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"무엇을 하셨어요?"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"flag_concern","input":{}}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"type\":\"OTHER_ANOMALY\","}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"reason\":\"집을 못 찾겠다고 하심\"}"}}`,
		`{"type":"content_block_stop","index":1}`,
		`{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":42}}`,
		`{"type":"message_stop"}`,
	}
	for _, e := range events {
		var typ struct{ Type string }
		_ = json.Unmarshal([]byte(e), &typ)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", typ.Type, e)
	}
}

func TestAnthropicStream(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(b, &body); err != nil {
			t.Errorf("request body: %v", err)
		}
		sse(w)
	}))
	defer srv.Close()

	c := NewAnthropic("test-key", "claude-sonnet-5-5", option.WithBaseURL(srv.URL), option.WithMaxRetries(0))
	var text strings.Builder
	res, err := c.Stream(context.Background(), Request{
		System: SystemPrompt("PICKUP_BRIDGE", "김순자", []string{"손주"}),
		Messages: []Message{
			{User, Kickoff()},
			{Assistant, "안녕하세요, 김순자님."},
			{User, "손주가 어제 왔어"},
		},
		Continuation: Continuation("아이고, 그러셨어요."),
	}, func(s string) { text.WriteString(s) })
	if err != nil {
		t.Fatal(err)
	}

	if got, want := text.String(), "손주분이 오셨군요. 무엇을 하셨어요?"; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
	if res.Model != "claude-sonnet-5-5" || res.InputTokens != 150 || res.OutputTokens != 42 || res.Refused {
		t.Errorf("result = %+v", res)
	}
	if len(res.Concerns) != 1 || res.Concerns[0].Type != "OTHER_ANOMALY" || res.Concerns[0].Reason != "집을 못 찾겠다고 하심" {
		t.Errorf("concerns = %+v", res.Concerns)
	}

	// What went over the wire.
	if body["stream"] != true || body["model"] != "claude-sonnet-5-5" {
		t.Errorf("stream/model = %v/%v", body["stream"], body["model"])
	}
	if th, _ := body["thinking"].(map[string]any); th["type"] != "between_tools" {
		t.Errorf("thinking = %v", body["thinking"])
	}
	system := body["system"].([]any)
	if len(system) != 2 || system[0].(map[string]any)["cache_control"] == nil || system[1].(map[string]any)["cache_control"] != nil {
		t.Errorf("system = %v", system)
	}
	msgs := body["messages"].([]any)
	last := msgs[len(msgs)-1].(map[string]any)["content"].([]any)
	if len(last) != 2 || last[0].(map[string]any)["text"] != "손주가 어제 왔어" ||
		!strings.Contains(last[1].(map[string]any)["text"].(string), "아이고, 그러셨어요.") {
		t.Errorf("last message = %v", last)
	}
	tools := body["tools"].([]any)
	if len(tools) != 1 || tools[0].(map[string]any)["name"] != "flag_concern" {
		t.Errorf("tools = %v", tools)
	}
	if tc, _ := body["tool_choice"].(map[string]any); tc["type"] != "auto" {
		t.Errorf("tool_choice = %v", body["tool_choice"])
	}
}

func TestAnthropicOtherModelKeepsDefaults(t *testing.T) {
	p := NewAnthropic("k", "claude-haiku-4-5-20251001").params(Request{Messages: []Message{{User, "안녕"}}})
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "between_tools") || strings.Contains(string(b), "effort") {
		t.Errorf("params = %s", b)
	}
}

func TestAnthropicRefusal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, e := range []string{
			`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-5-5","content":[],"stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
			`{"type":"message_delta","delta":{"stop_reason":"refusal"},"usage":{"output_tokens":0}}`,
			`{"type":"message_stop"}`,
		} {
			var typ struct{ Type string }
			_ = json.Unmarshal([]byte(e), &typ)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", typ.Type, e)
		}
	}))
	defer srv.Close()
	c := NewAnthropic("k", "claude-sonnet-5-5", option.WithBaseURL(srv.URL), option.WithMaxRetries(0))
	res, err := c.Stream(context.Background(), Request{Messages: []Message{{User, "안녕"}}}, func(string) {})
	if err != nil || !res.Refused {
		t.Fatalf("res = %+v, err = %v; want refused", res, err)
	}
}
