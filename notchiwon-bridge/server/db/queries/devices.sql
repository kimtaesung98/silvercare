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

-- name: CreateElderTablet :one
INSERT INTO device (kind, elder_id, token_hash, label)
VALUES ('ELDER_TABLET', sqlc.arg(elder_id), sqlc.arg(token_hash), sqlc.narg(label))
RETURNING *;

-- name: ListCaregiverPhones :many
-- 위급 알림을 보낼 조무사 휴대폰들.
SELECT * FROM device
WHERE kind = 'CAREGIVER_PHONE'
  AND caregiver_id = $1
  AND revoked_at IS NULL
  AND fcm_token IS NOT NULL;

-- name: RevokeDeviceByFcmToken :exec
-- FCM이 등록되지 않은 토큰이라고 답하면 다시 보내지 않습니다.
UPDATE device SET revoked_at = now() WHERE fcm_token = $1 AND revoked_at IS NULL;
