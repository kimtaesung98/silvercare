-- name: CreateCaregiverLocation :exec
INSERT INTO caregiver_location (caregiver_id, visit_id, latitude, longitude, accuracy_m, eta_minutes, recorded_at)
VALUES (
    sqlc.arg(caregiver_id), sqlc.narg(visit_id), sqlc.arg(latitude), sqlc.arg(longitude),
    sqlc.narg(accuracy_m), sqlc.narg(eta_minutes), sqlc.arg(recorded_at)
);

-- name: DeleteCaregiverLocationsBefore :execrows
DELETE FROM caregiver_location WHERE received_at < $1;
