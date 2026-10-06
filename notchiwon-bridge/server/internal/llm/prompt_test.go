package llm

import (
	"regexp"
	"strings"
	"testing"
)

func TestSystemPrompt(t *testing.T) {
	sys := SystemPrompt("PICKUP_BRIDGE", "김순자", []string{"손주", "화투", "시장"})
	if len(sys) != 2 {
		t.Fatalf("blocks = %d", len(sys))
	}
	if strings.Contains(sys[0], "김순자") {
		t.Error("cached block must not depend on the elder")
	}
	if !strings.Contains(sys[0], "조무사 선생님이 도착하기 전까지") {
		t.Errorf("pickup role missing: %s", sys[0])
	}
	for _, want := range []string{"김순자님", "손주, 화투, 시장"} {
		if !strings.Contains(sys[1], want) {
			t.Errorf("profile block %q lacks %q", sys[1], want)
		}
	}

	companion := SystemPrompt("COMPANION", "김순자", nil)
	if !strings.Contains(companion[0], "말동무") {
		t.Errorf("companion role missing: %s", companion[0])
	}
	if !strings.Contains(companion[1], "아직 등록된 관심사가 없으니") {
		t.Errorf("no-keyword profile: %s", companion[1])
	}
}

func TestFixedPrompts(t *testing.T) {
	// docs/specs/conversation-loop.md section 2.
	if got := Kickoff(); got != "대화를 시작해줘. 노인에게 먼저 친근하게 인사하고 관심사 화제를 자연스럽게 꺼내줘." {
		t.Errorf("Kickoff() = %q", got)
	}
	if got := Continuation("아이고, 그러셨어요."); !strings.Contains(got, `"아이고, 그러셨어요."`) {
		t.Errorf("Continuation() = %q", got)
	}
	if !regexp.MustCompile(`^conv-[0-9a-f]{10}$`).MatchString(PromptVersion) {
		t.Errorf("PromptVersion = %q", PromptVersion)
	}
}
