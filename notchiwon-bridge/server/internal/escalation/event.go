package escalation

import (
	"context"

	"github.com/google/uuid"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
)

// Sources of escalation_event.source.
const (
	SourceRule = "RULE"
	SourceLLM  = "LLM"
)

// FromRule records the event a rule match raises on an elder utterance.
func FromRule(ctx context.Context, q *db.Queries, sessionID, utteranceID uuid.UUID, m Match) (db.EscalationEvent, error) {
	return q.CreateEscalationEvent(ctx, db.CreateEscalationEventParams{
		SessionID:       sessionID,
		UtteranceID:     &utteranceID,
		TriggerType:     string(m.Type),
		Source:          SourceRule,
		RuleID:          &m.RuleID,
		NotifiedTargets: noTargets,
	})
}

// FromLLM records a concern Claude reported with the flag_concern tool. It is
// a second line of defence behind the rules, never a replacement for them.
func FromLLM(ctx context.Context, q *db.Queries, sessionID, utteranceID uuid.UUID, t TriggerType, reason string) (db.EscalationEvent, error) {
	if !t.Valid() {
		t = OtherAnomaly
	}
	return q.CreateEscalationEvent(ctx, db.CreateEscalationEventParams{
		SessionID:       sessionID,
		UtteranceID:     &utteranceID,
		TriggerType:     string(t),
		Source:          SourceLLM,
		Reason:          &reason,
		NotifiedTargets: noTargets,
	})
}

// noTargets is notified_targets until FCM alerts arrive in stage 5.
var noTargets = []byte(`[]`)
