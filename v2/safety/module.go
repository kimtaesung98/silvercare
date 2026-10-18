package safety

import (
	"context"
	"log/slog"

	"github.com/kimtaesung98/silvercare/v2/chat"
)

// 메시지에 붙는 표시.
const (
	Name    = "safety"
	TagType = "safety"      // Type 값
	TagRule = "safety_rule" // Rule.ID
)

// ReassureText는 위급한 말에 AI 대신 보내는 안심 문장입니다.
const ReassureText = "지금 많이 힘드시죠. 곧 도와드릴 분께 알렸어요. 조금만 기다려 주세요."

// Module은 chat.Module입니다. 사람이 쓴 메시지만 검사합니다.
type Module struct{}

func (Module) Name() string { return Name }

func (Module) Inspect(m *chat.Message) {
	if m.From.Kind != chat.Human {
		return
	}
	match, ok := Check(m.Text)
	if !ok {
		return
	}
	if m.Tags == nil {
		m.Tags = map[string]string{}
	}
	m.Tags[TagType] = string(match.Type)
	m.Tags[TagRule] = match.RuleID
	m.Tags[chat.TagHandledBy] = Name
}

// Guard는 위급 표시가 붙은 메시지에 안심 문장으로 답하는 참가자입니다.
// 표시가 붙은 메시지 자체가 방에 있는 사람들에게 가는 알림이고, 화면이 그 표시를 보여 줍니다.
type Guard struct {
	Hub *chat.Hub
	Log *slog.Logger
}

// Join은 안전 지킴이를 방에 넣고 ctx가 끝날 때까지 지킵니다.
func (g *Guard) Join(ctx context.Context, roomID string) error {
	self := chat.Participant{ID: "safety-" + roomID, Name: "안전 지킴이", Kind: chat.System}
	if err := g.Hub.Join(roomID, self); err != nil {
		return err
	}
	sub, err := g.Hub.Subscribe(roomID)
	if err != nil {
		return err
	}
	log := g.Log
	if log == nil {
		log = slog.Default()
	}
	go func() {
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-sub.C():
				if !ok {
					return
				}
				if m.Tags[TagType] == "" {
					continue
				}
				log.Warn("위급 감지", "room", roomID, "from", m.From.Name, "type", m.Tags[TagType], "rule", m.Tags[TagRule])
				if _, err := g.Hub.Post(roomID, self.ID, ReassureText); err != nil {
					log.Warn("safety: post", "room", roomID, "err", err)
				}
			}
		}
	}()
	return nil
}
