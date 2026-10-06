// Package opener picks sentence 0 of an AI turn: a short, pre-recorded
// back-channel line the tablet plays at once while Claude is still writing
// sentence 1. Picking it uses rules only, never an LLM (architecture.md 6.2).
package opener

import (
	"regexp"
	"strings"
	"unicode"
)

// Category is opener_clip.category.
type Category string

// Categories. Classify returns the first five; Escalation and Filler are
// chosen by the conversation engine.
const (
	Recall      Category = "RECALL"
	Question    Category = "QUESTION"
	Emotion     Category = "EMOTION"
	ShortAnswer Category = "SHORT_ANSWER"
	Greeting    Category = "GREETING"
	Escalation  Category = "ESCALATION"
	Filler      Category = "FILLER"
)

var (
	greetingRe = regexp.MustCompile(`^(안녕|반가|어서 ?와|잘 ?(잤|있었|지냈))`)
	// Question words, or an ending that asks. Spoken questions often lose
	// their question mark in STT, so the ending matters as much as "?".
	questionRe = regexp.MustCompile(`[?？]|(뭐|무엇|무슨|어디|언제|누구|누가|왜|어떻게|어째서)|(니|냐|나요|까|까요)$`)
	emotionRe  = regexp.MustCompile(`슬프|슬퍼|외로|무서|무섭|걱정|속상|서운|섭섭|보고 ?싶|그립|그리워|기쁘|기뻐|좋아|행복|심심|답답|화가|화나|짜증|눈물|울었|울고|우울|불안|고마|미안`)
)

// shortAnswerRunes is the longest reply, in letters, that counts as a short answer ("응", "그래", "아니야").
const shortAnswerRunes = 4

// Classify sorts an elder utterance into one of the opener categories.
// The order matters: a greeting or question that is also short keeps its own category.
func Classify(text string) Category {
	t := strings.TrimSpace(text)
	trimmed := strings.TrimRightFunc(t, func(r rune) bool { return unicode.IsPunct(r) && r != '?' && r != '？' || unicode.IsSpace(r) })
	switch {
	case greetingRe.MatchString(t):
		return Greeting
	case questionRe.MatchString(trimmed):
		return Question
	case emotionRe.MatchString(t):
		return Emotion
	case letters(t) <= shortAnswerRunes:
		return ShortAnswer
	default:
		return Recall
	}
}

func letters(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			n++
		}
	}
	return n
}
