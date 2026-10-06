// Package escalation is the safety layer: the keyword rules every elder
// utterance passes before Claude sees it, and the escalation events they raise.
package escalation

import (
	"regexp"
	"strings"
)

// TriggerType is escalation_event.trigger_type.
type TriggerType string

// Trigger types. OtherAnomaly comes only from Claude's flag_concern tool.
const (
	FallMention     TriggerType = "FALL_MENTION"
	PainComplaint   TriggerType = "PAIN_COMPLAINT"
	SelfOrOtherHarm TriggerType = "SELF_OR_OTHER_HARM"
	OtherAnomaly    TriggerType = "OTHER_ANOMALY"
)

// Valid reports whether t is a known trigger type.
func (t TriggerType) Valid() bool {
	switch t {
	case FallMention, PainComplaint, SelfOrOtherHarm, OtherAnomaly:
		return true
	}
	return false
}

// Rule is one keyword pattern. ID is stored as escalation_event.rule_id so a
// later review can see which pattern fired; never reuse an ID for a new pattern.
type Rule struct {
	ID      string
	Type    TriggerType
	Pattern *regexp.Regexp
}

// Rules are checked in order and the first match wins
// (docs/specs/conversation-loop.md section 3). The list is a draft until the
// care experts review it in stage 7.
var Rules = []Rule{
	{"fall.fell", FallMention, regexp.MustCompile(`넘어졌`)},
	{"fall.collapsed", FallMention, regexp.MustCompile(`쓰러졌`)},
	{"fall.word", FallMention, regexp.MustCompile(`낙상`)},
	{"pain.apa", PainComplaint, regexp.MustCompile(`아파`)},
	{"pain.apeu", PainComplaint, regexp.MustCompile(`아프`)},
	{"pain.word", PainComplaint, regexp.MustCompile(`통증`)},
	{"pain.breath", PainComplaint, regexp.MustCompile(`숨이\s*차`)},
	{"harm.want_die", SelfOrOtherHarm, regexp.MustCompile(`죽고\s*싶`)},
	{"harm.self", SelfOrOtherHarm, regexp.MustCompile(`자해`)},
	{"harm.hit", SelfOrOtherHarm, regexp.MustCompile(`때리`)},
}

// Match is the rule an utterance tripped.
type Match struct {
	RuleID string
	Type   TriggerType
}

// harmless are words that contain a rule pattern but mean nothing worrying.
// RE2 has no negative lookahead, so they are blanked out before matching.
var harmless = strings.NewReplacer("아파트", " ")

// Check runs the rules over text and returns the first match.
func Check(text string) (Match, bool) {
	text = harmless.Replace(text)
	for _, r := range Rules {
		if r.Pattern.MatchString(text) {
			return Match{RuleID: r.ID, Type: r.Type}, true
		}
	}
	return Match{}, false
}
