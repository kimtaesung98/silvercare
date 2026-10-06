package session

import "strings"

// stopPhrases end a companion conversation when the elder says them. They
// are checked after the safety rules, so "죽고 싶어, 그만할래" still escalates.
var stopPhrases = []string{
	"그만할게", "그만 할게", "그만할래", "그만 할래", "그만하자", "그만 하자",
	"그만 얘기", "그만 이야기", "이제 그만", "나중에 하자", "나중에 얘기",
	"이제 잘게", "먼저 잘게", "잘래", "자야겠", "자러 갈", "들어가 볼게", "끊을게", "다음에 하자",
}

// WantsToStop reports whether an utterance asks to end the conversation.
func WantsToStop(text string) bool {
	t := strings.ReplaceAll(strings.TrimSpace(text), "  ", " ")
	for _, p := range stopPhrases {
		if strings.Contains(t, p) {
			return true
		}
	}
	return false
}
