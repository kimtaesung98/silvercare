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

-- name: GetEscalationDetail :one
-- 조무사 앱 화면용: 어르신 이름, 감지된 발화, 방문(누구의 방문인지 확인용)을 함께 읽습니다.
-- 말동무 세션의 위급은 방문이 없으므로 visit 열이 NULL입니다.
SELECT
    sqlc.embed(escalation_event),
    elder.id AS elder_id,
    elder.name AS elder_name,
    utterance.text AS utterance_text,
    visit.id AS visit_id,
    visit.caregiver_id AS caregiver_id
FROM escalation_event
JOIN conversation_session ON conversation_session.id = escalation_event.session_id
JOIN elder ON elder.id = conversation_session.elder_id
LEFT JOIN utterance ON utterance.id = escalation_event.utterance_id
LEFT JOIN visit ON visit.id = conversation_session.visit_id
WHERE escalation_event.id = $1;

-- name: ListOpenEscalationsForCaregiver :many
-- 아직 확인하지 않은 위급. 푸시를 놓쳤을 때 앱이 첫 화면에서 다시 보여줍니다.
SELECT
    sqlc.embed(escalation_event),
    elder.id AS elder_id,
    elder.name AS elder_name,
    utterance.text AS utterance_text,
    visit.id AS visit_id,
    visit.caregiver_id AS caregiver_id
FROM escalation_event
JOIN conversation_session ON conversation_session.id = escalation_event.session_id
JOIN elder ON elder.id = conversation_session.elder_id
JOIN visit ON visit.id = conversation_session.visit_id
LEFT JOIN utterance ON utterance.id = escalation_event.utterance_id
WHERE visit.caregiver_id = sqlc.arg(caregiver_id)
  AND escalation_event.acknowledged_at IS NULL
  AND escalation_event.created_at >= sqlc.arg(since)
ORDER BY escalation_event.created_at DESC;

-- name: SetEscalationNotified :exec
-- 알림을 보낸 대상(기기 ID 목록)을 남깁니다.
UPDATE escalation_event SET notified_targets = sqlc.arg(notified_targets) WHERE id = sqlc.arg(id);


-- name: ListSessionEscalations :many
SELECT * FROM escalation_event WHERE session_id = $1 ORDER BY created_at;

-- name: ListElderEscalationsBetween :many
-- 하루 요약용: 그날 말동무 대화에서 감지한 위급.
SELECT escalation_event.*
FROM escalation_event
JOIN conversation_session ON conversation_session.id = escalation_event.session_id
WHERE conversation_session.elder_id = sqlc.arg(elder_id)
  AND conversation_session.mode = 'COMPANION'
  AND escalation_event.created_at >= sqlc.arg(from_time)
  AND escalation_event.created_at < sqlc.arg(to_time)
ORDER BY escalation_event.created_at;
