package session

import (
	"slices"
	"testing"
)

func TestSplitter(t *testing.T) {
	cases := []struct {
		name   string
		chunks []string
		want   []string
	}{
		{"korean endings", []string{"손주분이 오셨군요. 무엇을 하셨어요? 좋으셨겠어요!"}, []string{"손주분이 오셨군요.", "무엇을 하셨어요?", "좋으셨겠어요!"}},
		{"split across deltas", []string{"날씨가 참 좋", "네요.", " 산책 ", "가실래요?"}, []string{"날씨가 참 좋네요.", "산책 가실래요?"}},
		{"decimal and ellipsis stay", []string{"1.5킬로미터 정도요... 가까워요. 끝"}, []string{"1.5킬로미터 정도요...", "가까워요.", "끝"}},
		{"closing quote", []string{`"그렇지요." 하셨군요. `}, []string{`"그렇지요."`, "하셨군요."}},
		{"newline", []string{"첫 줄\n둘째 줄"}, []string{"첫 줄", "둘째 줄"}},
		{"no terminator", []string{"그렇군요"}, []string{"그렇군요"}},
		{"blank", []string{"  ", "\n"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var s splitter
			var got []string
			for _, c := range tc.chunks {
				got = append(got, s.Push(c)...)
			}
			if rest := s.Flush(); rest != "" {
				got = append(got, rest)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// A sentence is released as soon as the space after it arrives, before the stream ends.
func TestSplitterReleasesEarly(t *testing.T) {
	var s splitter
	if got := s.Push("안녕하세요."); got != nil {
		t.Fatalf("released before the space: %q", got)
	}
	if got := s.Push(" 오늘"); !slices.Equal(got, []string{"안녕하세요."}) {
		t.Fatalf("got %q", got)
	}
}
