package safety

import (
	"context"
	"testing"
	"time"

	"github.com/kimtaesung98/silvercare/v2/chat"
)

func TestCheckDetects(t *testing.T) {
	for _, tc := range []struct {
		text string
		want Type
	}{
		{"아까 화장실에서 넘어졌어", Fall},
		{"넘어져서 못 일어나", Fall},
		{"길에서 쓰러졌었지", Fall},
		{"어지러워서 쓰러질 것 같아", Fall},
		{"작년에 낙상 사고가 있었어", Fall},
		{"다리가 너무 아파", Pain},
		{"다리가 아퍼", Pain},
		{"어제 많이 아팠어", Pain},
		{"머리가 아프네", Pain},
		{"허리 통증이 있어", Pain},
		{"숨이차", Pain},
		{"가슴이 답답해", Pain},
		{"우리 아파트 계단 오르면 다리가 아파", Pain},
		{"그냥 죽고 싶어", Harm},
		{"살기 싫어", Harm},
		{"누가 나를 때리려고 해", Harm},
	} {
		m, ok := Check(tc.text)
		if !ok || m.Type != tc.want {
			t.Errorf("Check(%q) = %+v, %v; want %s", tc.text, m, ok, tc.want)
		}
	}
}

func TestCheckIgnores(t *testing.T) {
	for _, text := range []string{
		"오늘 날씨가 좋네",
		"손주가 어제 놀러 왔어",
		"우리 아파트 앞에 꽃이 폈어",
		"드라마에서 때리더라",
		"",
	} {
		if m, ok := Check(text); ok {
			t.Errorf("Check(%q) = %+v, want no match", text, m)
		}
	}
}

func TestRuleIDsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range Rules {
		if seen[r.ID] {
			t.Errorf("duplicate %s", r.ID)
		}
		seen[r.ID] = true
	}
}

func TestModuleTagsAndGuardReassures(t *testing.T) {
	h := chat.NewHub()
	h.Use(Module{})
	room := h.CreateRoom()
	_ = h.Join(room, chat.Participant{ID: "e", Name: "어르신", Kind: chat.Human})
	_ = h.Join(room, chat.Participant{ID: "ai", Name: "말동무", Kind: chat.AI})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := (&Guard{Hub: h}).Join(ctx, room); err != nil {
		t.Fatal(err)
	}
	sub, _ := h.Subscribe(room)
	defer sub.Close()

	// AI가 쓴 말은 검사하지 않음
	ai, _ := h.Post(room, "ai", "어디가 아프세요?")
	if ai.Tags != nil {
		t.Fatalf("ai tagged %+v", ai.Tags)
	}
	m, _ := h.Post(room, "e", "다리가 너무 아파")
	if m.Tags[TagType] != string(Pain) || m.Tags[TagRule] != "pain.apa" || m.Tags[chat.TagHandledBy] != Name {
		t.Fatalf("tags %+v", m.Tags)
	}
	<-sub.C()
	<-sub.C()
	select {
	case r := <-sub.C():
		if r.From.Kind != chat.System || r.Text != ReassureText {
			t.Fatalf("got %+v", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no reassurance")
	}
}
