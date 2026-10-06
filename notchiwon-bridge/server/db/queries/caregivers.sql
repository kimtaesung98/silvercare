-- name: GetCaregiver :one
SELECT * FROM caregiver WHERE id = $1;

-- name: GetCaregiverByLoginID :one
SELECT * FROM caregiver WHERE login_id = $1;

-- name: SetCaregiverLogin :one
UPDATE caregiver
SET login_id = sqlc.arg(login_id),
    password_hash = sqlc.arg(password_hash)
WHERE id = sqlc.arg(id)
RETURNING *;
