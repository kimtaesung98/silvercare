// Package safety는 위급 필터 모듈입니다. 사람이 쓴 메시지를 AI보다 먼저 보고,
// 위급한 말이면 표시를 붙여 AI가 스스로 답하지 않게 하고, 안전 지킴이가
// 안심 문장을 보내 방에 있는 사람들이 바로 알게 합니다.
package safety

import (
	"regexp"
	"strings"
)

// Type은 위급의 종류입니다. 화면은 이 값으로 이름을 보여 줍니다.
type Type string

const (
	Fall Type = "fall" // 낙상
	Pain Type = "pain" // 통증·몸 이상
	Harm Type = "harm" // 자해·타해
)

// Rule은 패턴 하나입니다. ID는 나중에 어떤 규칙이 걸렸는지 보려고 남기며,
// 한 번 쓴 ID를 다른 패턴에 다시 쓰지 않습니다.
type Rule struct {
	ID      string
	Type    Type
	Pattern *regexp.Regexp
}

// Rules는 순서대로 검사하고 처음 걸린 규칙을 씁니다. 이전 버전의 규칙에
// 구어 활용형(이전 핸드북 CR-01)을 더했습니다. 요양·의료 전문가 검토 전 초안입니다.
var Rules = []Rule{
	{"fall.fell", Fall, regexp.MustCompile(`넘어[졌져지질]`)},
	{"fall.collapsed", Fall, regexp.MustCompile(`쓰러[졌져지질]`)},
	{"fall.word", Fall, regexp.MustCompile(`낙상`)},
	{"fall.dizzy", Fall, regexp.MustCompile(`어지러`)},
	{"fall.cant_stand", Fall, regexp.MustCompile(`못\s*일어나`)},
	{"pain.apa", Pain, regexp.MustCompile(`아[파퍼팠프픈]`)},
	{"pain.word", Pain, regexp.MustCompile(`통증`)},
	{"pain.breath", Pain, regexp.MustCompile(`숨이\s*차|숨을?\s*못`)},
	{"pain.chest", Pain, regexp.MustCompile(`답답해`)},
	{"harm.want_die", Harm, regexp.MustCompile(`죽고\s*싶|살기\s*싫`)},
	{"harm.self", Harm, regexp.MustCompile(`자해`)},
	{"harm.hit", Harm, regexp.MustCompile(`나를?\s*때리|날\s*때리|때려요|때렸어`)},
}

// harmless는 규칙 글자를 품었지만 걱정할 뜻이 없는 말입니다. Go 정규식에는
// 부정 전방 탐색이 없어 미리 지웁니다.
var harmless = strings.NewReplacer("아파트", " ")

// Match는 걸린 규칙입니다.
type Match struct {
	RuleID string
	Type   Type
}

// Check는 text에 규칙을 돌려 처음 걸린 것을 돌려줍니다.
func Check(text string) (Match, bool) {
	text = harmless.Replace(text)
	for _, r := range Rules {
		if r.Pattern.MatchString(text) {
			return Match{RuleID: r.ID, Type: r.Type}, true
		}
	}
	return Match{}, false
}
