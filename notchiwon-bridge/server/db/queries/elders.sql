-- name: GetElder :one
SELECT * FROM elder WHERE id = $1;

-- name: GetCompanionSchedule :one
SELECT * FROM companion_schedule WHERE elder_id = $1;

-- name: UpsertCompanionSchedule :one
INSERT INTO companion_schedule (elder_id, enabled, check_in_times, bedtime_start, bedtime_end, time_zone, daily_token_limit)
VALUES (
    sqlc.arg(elder_id), sqlc.arg(enabled), sqlc.arg(check_in_times), sqlc.narg(bedtime_start),
    sqlc.narg(bedtime_end), sqlc.arg(time_zone), sqlc.narg(daily_token_limit)
)
ON CONFLICT (elder_id) DO UPDATE
SET enabled = EXCLUDED.enabled,
    check_in_times = EXCLUDED.check_in_times,
    bedtime_start = EXCLUDED.bedtime_start,
    bedtime_end = EXCLUDED.bedtime_end,
    time_zone = EXCLUDED.time_zone,
    daily_token_limit = EXCLUDED.daily_token_limit
RETURNING *;

-- name: GetDailyDigest :one
SELECT * FROM daily_digest WHERE elder_id = $1 AND digest_date = $2;

-- name: ListEnabledCompanionSchedules :many
SELECT * FROM companion_schedule WHERE enabled;

-- name: GetElderGuardian :one
SELECT
    elder.id AS elder_id,
    elder.name AS elder_name,
    elder.center_id,
    guardian.id AS guardian_id,
    guardian.name AS guardian_name,
    guardian.phone AS guardian_phone
FROM elder
JOIN guardian ON guardian.id = elder.guardian_id
WHERE elder.id = $1;

-- name: InsertDailyDigest :one
-- 같은 날 요약이 이미 있으면 행이 없습니다(재시도해도 문자는 한 번).
INSERT INTO daily_digest (elder_id, digest_date, summary_text, emotion_flag, session_count, escalation_count, total_tokens, model)
VALUES (
    sqlc.arg(elder_id), sqlc.arg(digest_date), sqlc.arg(summary_text), sqlc.narg(emotion_flag),
    sqlc.arg(session_count), sqlc.arg(escalation_count), sqlc.arg(total_tokens), sqlc.narg(model)
)
ON CONFLICT (elder_id, digest_date) DO NOTHING
RETURNING *;

-- name: MarkDailyDigestSent :exec
UPDATE daily_digest SET sent_at = now() WHERE id = $1 AND sent_at IS NULL;
