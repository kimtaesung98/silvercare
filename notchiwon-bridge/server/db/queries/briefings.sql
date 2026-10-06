-- name: GetBriefingBySession :one
SELECT * FROM briefing_report WHERE session_id = $1;

-- name: MarkBriefingRead :one
UPDATE briefing_report
SET read_by_caregiver_at = COALESCE(read_by_caregiver_at, now())
WHERE session_id = $1
RETURNING *;
