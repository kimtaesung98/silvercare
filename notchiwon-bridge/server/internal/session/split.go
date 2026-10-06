package session

import (
	"strings"
	"unicode"
)

// splitter cuts streamed text into sentences so each can go to TTS and the
// tablet as soon as it is complete (architecture.md 6.1 step 6).
//
// A sentence ends at a run of . ? ! … (Korean replies end in 다. 요. ? !),
// optionally followed by closing quotes or brackets, once whitespace follows.
// Waiting for the whitespace keeps "1.5" and "..." in one piece. A newline also
// ends a sentence.
type splitter struct {
	buf []rune
}

// Push adds streamed text and returns the sentences it completed.
func (s *splitter) Push(text string) []string {
	s.buf = append(s.buf, []rune(text)...)
	var out []string
	start := 0
	for i := 0; i < len(s.buf); i++ {
		r := s.buf[i]
		if r == '\n' {
			out = appendSentence(out, s.buf[start:i])
			start = i + 1
			continue
		}
		if !unicode.IsSpace(r) || i == start {
			continue
		}
		// r is whitespace: does the text before it end a sentence?
		j := i - 1
		for j >= start && isCloser(s.buf[j]) {
			j--
		}
		if j >= start && isTerminator(s.buf[j]) {
			out = appendSentence(out, s.buf[start:i])
			start = i + 1
		}
	}
	s.buf = append(s.buf[:0], s.buf[start:]...)
	return out
}

// Flush returns what is left once the stream ends.
func (s *splitter) Flush() string {
	rest := strings.TrimSpace(string(s.buf))
	s.buf = s.buf[:0]
	return rest
}

func appendSentence(out []string, r []rune) []string {
	if t := strings.TrimSpace(string(r)); t != "" {
		out = append(out, t)
	}
	return out
}

func isTerminator(r rune) bool {
	switch r {
	case '.', '?', '!', '…', '。', '？', '！':
		return true
	}
	return false
}

func isCloser(r rune) bool {
	switch r {
	case '"', '\'', '”', '’', ')', '」', '』', '~':
		return true
	}
	return false
}
