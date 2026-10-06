-- name: CreateEscalationEvent :one
INSERT INTO escalation_event (session_id, utterance_id, trigger_type, source, rule_id, reason, notified_targets)
VALUES (
    sqlc.arg(session_id), sqlc.narg(utterance_id), sqlc.arg(trigger_type), sqlc.arg(source),
    sqlc.narg(rule_id), sqlc.narg(reason), sqlc.arg(notified_targets)
)
RETURNING *;

-- name: GetEscalationEvent :one
SELECT * FROM escalation_event WHERE id = $1;

-- name: AcknowledgeEscalation :one
-- 여러 번 눌러도 처음 확인한 시각과 사람이 남습니다.
UPDATE escalation_event
SET acknowledged_at = COALESCE(acknowledged_at, now()),
    acknowledged_by_caregiver_id = COALESCE(acknowledged_by_caregiver_id, sqlc.arg(caregiver_id))
WHERE id = sqlc.arg(id)
RETURNING *;
