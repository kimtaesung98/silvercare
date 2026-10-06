package opener

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		text string
		want Category
	}{
		{"안녕하세요", Greeting},
		{"반가워", Greeting},
		{"잘 잤어", Greeting},
		{"선생님은 언제 와?", Question},
		{"오늘 점심은 뭐야", Question},
		{"그게 무슨 말이니", Question},
		{"우리 아들 왔나요", Question},
		{"응?", Question},
		{"요즘 너무 외로워", Emotion},
		{"손주가 보고 싶어", Emotion},
		{"오늘 기분이 좋아", Emotion},
		{"응", ShortAnswer},
		{"그래.", ShortAnswer},
		{"아니야", ShortAnswer},
		{"손주가 어제 놀러 왔어", Recall},
		{"옛날에 시장에서 떡을 팔았지", Recall},
		{"우리 아파트 앞에 꽃이 폈어", Recall},
	}
	for _, tc := range cases {
		if got := Classify(tc.text); got != tc.want {
			t.Errorf("Classify(%q) = %s, want %s", tc.text, got, tc.want)
		}
	}
}
