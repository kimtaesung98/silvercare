-- name: GetGuardian :one
SELECT * FROM guardian WHERE id = $1;

-- name: GetGuardianByLoginID :one
SELECT * FROM guardian WHERE login_id = $1;

-- name: SetGuardianLogin :one
UPDATE guardian
SET login_id = sqlc.arg(login_id),
    password_hash = sqlc.arg(password_hash)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ListGuardianElders :many
-- 보호자가 돌보는 어르신. 보호자 앱의 모든 조회는 이 목록 안으로 제한합니다.
SELECT id, name FROM elder WHERE guardian_id = $1 ORDER BY name;

-- name: RegisterGuardianPhone :one
-- 같은 FCM 토큰이 다시 오면 보호자와 이름만 갱신합니다(기기 초기화·재설치).
INSERT INTO device (kind, guardian_id, fcm_token, label, last_seen_at)
VALUES ('GUARDIAN_PHONE', sqlc.arg(guardian_id), sqlc.arg(fcm_token), sqlc.narg(label), now())
ON CONFLICT (fcm_token) DO UPDATE
SET kind = 'GUARDIAN_PHONE',
    guardian_id = EXCLUDED.guardian_id,
    caregiver_id = NULL,
    elder_id = NULL,
    label = EXCLUDED.label,
    revoked_at = NULL,
    last_seen_at = now()
RETURNING *;

-- name: ListGuardianPhones :many
SELECT * FROM device
WHERE kind = 'GUARDIAN_PHONE'
  AND guardian_id = $1
  AND revoked_at IS NULL
  AND fcm_token IS NOT NULL;

-- name: ListRecentDigests :many
-- 보호자 앱 홈: 어르신의 최근 하루 요약.
SELECT * FROM daily_digest
WHERE elder_id = sqlc.arg(elder_id)
ORDER BY digest_date DESC
LIMIT sqlc.arg(max_count);

-- name: ListOpenEscalationsForGuardian :many
-- 보호자가 확인하지 않은 위급. 말동무 세션(방문 없는 대화)만 보호자에게 보입니다.
SELECT
    sqlc.embed(escalation_event),
    elder.id AS elder_id,
    elder.name AS elder_name,
    utterance.text AS utterance_text
FROM escalation_event
JOIN conversation_session ON conversation_session.id = escalation_event.session_id
JOIN elder ON elder.id = conversation_session.elder_id
LEFT JOIN utterance ON utterance.id = escalation_event.utterance_id
WHERE elder.guardian_id = sqlc.arg(guardian_id)
  AND conversation_session.mode = 'COMPANION'
  AND escalation_event.acknowledged_at IS NULL
  AND escalation_event.created_at >= sqlc.arg(since)
ORDER BY escalation_event.created_at DESC;

-- name: AcknowledgeEscalationByGuardian :one
-- 여러 번 눌러도 처음 확인한 시각과 사람이 남습니다.
UPDATE escalation_event
SET acknowledged_at = COALESCE(acknowledged_at, now()),
    acknowledged_by_guardian_id = COALESCE(acknowledged_by_guardian_id, sqlc.arg(guardian_id))
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetGuardianEscalation :one
-- 보호자가 받은 알림을 열 때. 다른 집 어르신의 위급이면 행이 없습니다.
SELECT
    sqlc.embed(escalation_event),
    elder.id AS elder_id,
    elder.name AS elder_name,
    utterance.text AS utterance_text,
    conversation_session.mode AS session_mode
FROM escalation_event
JOIN conversation_session ON conversation_session.id = escalation_event.session_id
JOIN elder ON elder.id = conversation_session.elder_id
LEFT JOIN utterance ON utterance.id = escalation_event.utterance_id
WHERE escalation_event.id = sqlc.arg(id)
  AND elder.guardian_id = sqlc.arg(guardian_id);
