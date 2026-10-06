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
