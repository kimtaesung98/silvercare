-- name: GetActiveDeviceByTokenHash :one
SELECT * FROM device WHERE token_hash = $1 AND revoked_at IS NULL;

-- name: TouchDevice :exec
UPDATE device SET last_seen_at = now() WHERE id = $1;

-- name: RegisterCaregiverPhone :one
-- 같은 FCM 토큰이 다시 오면 주인을 바꾸고 되살립니다(앱 재설치·로그인 전환).
INSERT INTO device (kind, caregiver_id, fcm_token, label)
VALUES ('CAREGIVER_PHONE', sqlc.arg(caregiver_id), sqlc.arg(fcm_token), sqlc.narg(label))
ON CONFLICT (fcm_token) DO UPDATE
SET caregiver_id = EXCLUDED.caregiver_id,
    label = EXCLUDED.label,
    revoked_at = NULL
RETURNING *;
