// Package companion은 말동무 AI 참가자입니다. 사람 참가자와 똑같이 방에 들어와
// 메시지를 읽고 씁니다. 무엇을 말할지는 Brain이 정하므로, Brain만 바꾸면
// Claude, 다른 AI, 정해진 문장 중 무엇으로든 돌릴 수 있습니다.
package companion

import (
	"context"
	"log/slog"

	"github.com/kimtaesung98/silvercare/v2/chat"
)

// Brain은 지금까지의 대화를 보고 말동무가 할 다음 말을 정합니다.
type Brain interface {
	Reply(ctx context.Context, self chat.Participant, history []chat.Message) (string, error)
}

// FallbackText는 Brain이 실패했을 때 대화가 끊기지 않도록 하는 말입니다.
const FallbackText = "네, 듣고 있어요. 조금 더 이야기해 주세요."

// Companion은 방 하나에 들어간 말동무입니다.
type Companion struct {
	Hub   *chat.Hub
	Self  chat.Participant
	Brain Brain
	Log   *slog.Logger
	// AnswerOnly가 있으면 그 참가자의 말에만 답합니다 (예: 어르신). 비우면 모든 사람에게.
	AnswerOnly string
}

// Join은 말동무를 방에 넣고, ctx가 끝날 때까지 사람 메시지에 답합니다.
// 방에 들어가고 구독을 연 뒤에 돌아오므로, 그다음에 온 메시지는 놓치지 않습니다.
func (c *Companion) Join(ctx context.Context, roomID string) error {
	if err := c.Hub.Join(roomID, c.Self); err != nil {
		return err
	}
	sub, err := c.Hub.Subscribe(roomID)
	if err != nil {
		return err
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
				if !c.shouldAnswer(m) {
					continue
				}
				c.answer(ctx, roomID)
			}
		}
	}()
	return nil
}

// shouldAnswer: 사람이 쓴 말이고, 모듈이 맡지 않았고(예: 위급), 답할 상대일 때만.
func (c *Companion) shouldAnswer(m chat.Message) bool {
	if m.From.Kind != chat.Human || m.Tags[chat.TagHandledBy] != "" {
		return false
	}
	return c.AnswerOnly == "" || m.From.ID == c.AnswerOnly
}

func (c *Companion) answer(ctx context.Context, roomID string) {
	log := c.Log
	if log == nil {
		log = slog.Default()
	}
	history, err := c.Hub.History(roomID)
	if err != nil {
		log.Warn("companion: history", "room", roomID, "err", err)
		return
	}
	text, err := c.Brain.Reply(ctx, c.Self, history)
	if err != nil || text == "" {
		log.Warn("companion: brain failed, using fallback", "room", roomID, "err", err)
		text = FallbackText
	}
	if _, err := c.Hub.Post(roomID, c.Self.ID, text); err != nil {
		log.Warn("companion: post", "room", roomID, "err", err)
	}
}
