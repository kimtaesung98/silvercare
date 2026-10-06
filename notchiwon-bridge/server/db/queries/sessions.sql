-- name: CreatePickupSession :one
INSERT INTO conversation_session (visit_id, elder_id, mode, started_by, trigger_eta_minutes, prompt_version)
VALUES (sqlc.arg(visit_id), sqlc.arg(elder_id), 'PICKUP_BRIDGE', 'SYSTEM', sqlc.arg(trigger_eta_minutes), sqlc.narg(prompt_version))
RETURNING *;

-- name: CreateCompanionSession :one
-- 어르신당 진행 중 세션은 하나뿐이라(부분 유니크 인덱스) 이미 대화 중이면 유니크 위반으로 실패합니다.
INSERT INTO conversation_session (elder_id, mode, started_by, prompt_version)
VALUES (sqlc.arg(elder_id), 'COMPANION', sqlc.arg(started_by), sqlc.narg(prompt_version))
RETURNING *;

-- name: GetSession :one
SELECT * FROM conversation_session WHERE id = $1;

-- name: GetSessionByVisit :one
SELECT * FROM conversation_session WHERE visit_id = $1;

-- name: GetOpenSessionForElder :one
SELECT * FROM conversation_session WHERE elder_id = $1 AND ended_at IS NULL;

-- name: EndSession :one
-- 이미 끝난 세션이면 행이 없습니다.
UPDATE conversation_session
SET ended_at = now(),
    ended_reason = sqlc.arg(ended_reason)
WHERE id = sqlc.arg(id)
  AND ended_at IS NULL
RETURNING *;

-- name: PreemptCompanionSession :one
-- 픽업 대기 세션을 시작하기 전에 진행 중인 말동무 세션을 끝냅니다. 없으면 행이 없습니다.
UPDATE conversation_session
SET ended_at = now(),
    ended_reason = 'PREEMPTED'
WHERE elder_id = $1
  AND mode = 'COMPANION'
  AND ended_at IS NULL
RETURNING *;

-- name: SetSessionEmotionTag :exec
UPDATE conversation_session SET overall_emotion_tag = sqlc.arg(tag) WHERE id = sqlc.arg(id);

-- name: GetSessionOwner :one
-- 세션의 어르신과, 픽업 세션이면 그 방문의 조무사.
SELECT
    sqlc.embed(conversation_session),
    elder.name AS elder_name,
    visit.caregiver_id AS caregiver_id
FROM conversation_session
JOIN elder ON elder.id = conversation_session.elder_id
LEFT JOIN visit ON visit.id = conversation_session.visit_id
WHERE conversation_session.id = $1;

-- name: SetHistorySummary :exec
-- 더 앞선 요약으로 덮어쓰지 않습니다(작업이 늦게 끝나도 안전).
UPDATE conversation_session
SET history_summary = sqlc.arg(summary),
    history_summary_through = sqlc.arg(through)
WHERE id = sqlc.arg(id)
  AND (history_summary_through IS NULL OR history_summary_through < sqlc.arg(through));

-- name: ListOpenCompanionSessions :many
-- 무응답·취침 시간 종료를 확인할 진행 중 말동무 세션과 마지막 발화 시각.
SELECT
    sqlc.embed(conversation_session),
    COALESCE(
        (SELECT max(u.created_at) FROM utterance u WHERE u.session_id = conversation_session.id),
        conversation_session.started_at
    )::timestamptz AS last_activity_at
FROM conversation_session
WHERE mode = 'COMPANION' AND ended_at IS NULL;

-- name: HasScheduledSessionSince :one
-- 그 안부 시각에 이미 안부 대화를 시작했는지.
SELECT EXISTS (
    SELECT 1 FROM conversation_session
    WHERE elder_id = sqlc.arg(elder_id)
      AND started_by = 'SCHEDULE'
      AND started_at >= sqlc.arg(since)
) AS started;

-- name: ListElderSessionsBetween :many
-- 하루 요약용: 그날 끝난 말동무 세션과 브리핑(없으면 NULL).
SELECT
    sqlc.embed(conversation_session),
    briefing_report.summary_text,
    briefing_report.emotion_flag
FROM conversation_session
LEFT JOIN briefing_report ON briefing_report.session_id = conversation_session.id
WHERE conversation_session.elder_id = sqlc.arg(elder_id)
  AND conversation_session.mode = 'COMPANION'
  AND conversation_session.started_at >= sqlc.arg(from_time)
  AND conversation_session.started_at < sqlc.arg(to_time)
ORDER BY conversation_session.started_at;

-- name: ListEldersWithCompanionBetween :many
-- 하루 요약을 만들 어르신: 그날 말동무 대화가 있었던 어르신.
SELECT DISTINCT elder_id
FROM conversation_session
WHERE mode = 'COMPANION'
  AND started_at >= sqlc.arg(from_time)
  AND started_at < sqlc.arg(to_time);
