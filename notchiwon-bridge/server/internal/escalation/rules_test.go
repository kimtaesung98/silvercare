package escalation

import "testing"

// docs/specs/conversation-loop.md section 4.1.
func TestCheckDetects(t *testing.T) {
	cases := []struct {
		text string
		want TriggerType
	}{
		{"아까 화장실에서 넘어졌어", FallMention},
		{"길에서 쓰러졌었지", FallMention},
		{"작년에 낙상 사고가 있었어", FallMention},
		{"다리가 너무 아파", PainComplaint},
		{"머리가 아프네", PainComplaint},
		{"허리 통증이 있어", PainComplaint},
		{"숨이 차서 힘들어", PainComplaint},
		{"숨이차", PainComplaint},
		{"우리 아파트 계단 오르면 다리가 아파", PainComplaint},
		{"그냥 죽고 싶어", SelfOrOtherHarm},
		{"죽고싶다", SelfOrOtherHarm},
		{"자해를 했어", SelfOrOtherHarm},
		{"누가 나를 때리려고 해", SelfOrOtherHarm},
	}
	for _, tc := range cases {
		m, ok := Check(tc.text)
		if !ok {
			t.Errorf("Check(%q) found nothing, want %s", tc.text, tc.want)
			continue
		}
		if m.Type != tc.want {
			t.Errorf("Check(%q) = %s (%s), want %s", tc.text, m.Type, m.RuleID, tc.want)
		}
		if m.RuleID == "" {
			t.Errorf("Check(%q) has no rule id", tc.text)
		}
	}
}

func TestCheckIgnores(t *testing.T) {
	for _, text := range []string{
		"오늘 날씨가 좋네",
		"손주가 어제 놀러 왔어",
		"우리 아파트 앞에 꽃이 폈어",
		"아파트 단지에 새로 이사 왔어",
		"",
	} {
		if m, ok := Check(text); ok {
			t.Errorf("Check(%q) = %s (%s), want no match", text, m.Type, m.RuleID)
		}
	}
}

// The first rule in list order wins when several match.
func TestCheckFirstMatchWins(t *testing.T) {
	m, ok := Check("넘어져서 다리가 아파, 아니 넘어졌어")
	if !ok || m.Type != FallMention || m.RuleID != "fall.fell" {
		t.Fatalf("got %+v, %v; want fall.fell", m, ok)
	}
}

func TestRuleIDsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range Rules {
		if seen[r.ID] {
			t.Errorf("duplicate rule id %q", r.ID)
		}
		seen[r.ID] = true
		if !r.Type.Valid() || r.Type == OtherAnomaly {
			t.Errorf("rule %q has type %q", r.ID, r.Type)
		}
	}
}
