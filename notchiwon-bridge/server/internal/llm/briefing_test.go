package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"
)

func briefingServer(t *testing.T, stopReason, text string, body *map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if body != nil {
			if err := json.Unmarshal(b, body); err != nil {
				t.Errorf("request body: %v", err)
			}
		}
		resp, _ := json.Marshal(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-haiku-4-5",
			"content":     []any{map[string]any{"type": "text", "text": text}},
			"stop_reason": stopReason,
			"usage":       map[string]any{"input_tokens": 300, "output_tokens": 90},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

var briefingInput = BriefingInput{
	ElderName: "김순자",
	Lines: []Line{
		{Elder: false, Text: "안녕하세요, 김순자님."},
		{Elder: true, Text: "큰아들이 어제 왔어"},
		{Elder: false, Text: "아이고, 반가우셨겠어요."},
		{Elder: true, Text: "화투도 쳤지"},
	},
	Escalations: []string{"통증 호소: 다리가 아파"},
}

func TestBrief(t *testing.T) {
	var body map[string]any
	srv := briefingServer(t, "end_turn", `{
		"summary": "큰아들이 다녀간 이야기를 즐겁게 하심. 다리가 아프다고 한 번 말씀하심.",
		"keywords": [
			{"keyword": "큰아들", "category": "FAMILY", "emotionTone": "POSITIVE", "mentions": 2},
			{"keyword": " ", "category": "HOBBY", "emotionTone": "NEUTRAL", "mentions": 1},
			{"keyword": "화투", "category": "HOBBY", "emotionTone": "POSITIVE", "mentions": 0}
		],
		"emotionTag": "SLIGHTLY_ANXIOUS",
		"emotionFlag": "다리 통증 호소 1회"
	}`, &body)
	s := NewAnthropicSummarizer(NewAnthropic("k", "claude-haiku-4-5", option.WithBaseURL(srv.URL), option.WithMaxRetries(0)))

	b, err := s.Brief(context.Background(), briefingInput)
	if err != nil {
		t.Fatal(err)
	}
	if b.Model != "claude-haiku-4-5" || !strings.Contains(b.Summary, "큰아들") || b.EmotionTag != "SLIGHTLY_ANXIOUS" {
		t.Errorf("briefing = %+v", b)
	}
	if len(b.Keywords) != 2 || b.Keywords[0].Keyword != "큰아들" || b.Keywords[1].Mentions != 1 {
		t.Errorf("keywords = %+v (blank dropped, mentions at least 1)", b.Keywords)
	}
	if b.EmotionFlag == nil || *b.EmotionFlag != "다리 통증 호소 1회" {
		t.Errorf("emotionFlag = %v", b.EmotionFlag)
	}

	// The request asks for the JSON schema and carries the transcript.
	format := body["output_config"].(map[string]any)["format"].(map[string]any)
	if format["type"] != "json_schema" || format["schema"] == nil {
		t.Errorf("output_config.format = %v", format)
	}
	msgs, _ := json.Marshal(body["messages"])
	for _, want := range []string{"어르신: 큰아들이 어제 왔어", "AI: 아이고", "통증 호소: 다리가 아파"} {
		if !strings.Contains(string(msgs), want) {
			t.Errorf("transcript is missing %q: %s", want, msgs)
		}
	}
}

func TestBriefCapsKeywordsAndBlankFlag(t *testing.T) {
	kws := make([]string, 8)
	for i := range kws {
		kws[i] = `{"keyword":"k` + string(rune('a'+i)) + `","category":"HOBBY","emotionTone":"NEUTRAL","mentions":1}`
	}
	srv := briefingServer(t, "end_turn", `{"summary":"편안하셨음.","keywords":[`+strings.Join(kws, ",")+`],"emotionTag":"STABLE","emotionFlag":" "}`, nil)
	s := NewAnthropicSummarizer(NewAnthropic("k", "claude-haiku-4-5", option.WithBaseURL(srv.URL), option.WithMaxRetries(0)))
	b, err := s.Brief(context.Background(), briefingInput)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Keywords) != maxKeywords || b.EmotionFlag != nil {
		t.Errorf("keywords = %d, flag = %v", len(b.Keywords), b.EmotionFlag)
	}
}

func TestBriefFails(t *testing.T) {
	cases := map[string]struct{ stop, text string }{
		"refusal":    {"refusal", ""},
		"cut short":  {"max_tokens", `{"summary":"편안`},
		"not json":   {"end_turn", "요약입니다"},
		"no summary": {"end_turn", `{"summary":"","keywords":[],"emotionTag":"STABLE","emotionFlag":null}`},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			srv := briefingServer(t, c.stop, c.text, nil)
			s := NewAnthropicSummarizer(NewAnthropic("k", "claude-haiku-4-5", option.WithBaseURL(srv.URL), option.WithMaxRetries(0)))
			if _, err := s.Brief(context.Background(), briefingInput); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

func TestBriefingPromptVersionIsSeparate(t *testing.T) {
	if !strings.HasPrefix(BriefingPromptVersion, "brief-") || BriefingPromptVersion == PromptVersion {
		t.Errorf("BriefingPromptVersion = %q", BriefingPromptVersion)
	}
}
