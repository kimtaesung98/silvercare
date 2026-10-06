-- name: GetVisit :one
SELECT * FROM visit WHERE id = $1;

-- name: GetVisitDetail :one
-- API 응답용: 어르신 이름·댁 위치와 픽업 대기 세션 ID를 함께 읽습니다.
SELECT
    sqlc.embed(visit),
    elder.name AS elder_name,
    elder.home_address,
    elder.home_latitude,
    elder.home_longitude,
    conversation_session.id AS session_id
FROM visit
JOIN elder ON elder.id = visit.elder_id
LEFT JOIN conversation_session ON conversation_session.visit_id = visit.id
WHERE visit.id = $1;

-- name: ListCaregiverVisitsBetween :many
-- 조무사 앱의 오늘 방문 목록. [from, to) 구간은 호출하는 쪽이 한국 시각 하루로 계산합니다.
SELECT
    sqlc.embed(visit),
    elder.name AS elder_name,
    elder.home_address,
    elder.home_latitude,
    elder.home_longitude,
    conversation_session.id AS session_id
FROM visit
JOIN elder ON elder.id = visit.elder_id
LEFT JOIN conversation_session ON conversation_session.visit_id = visit.id
WHERE visit.caregiver_id = sqlc.arg(caregiver_id)
  AND visit.scheduled_time >= sqlc.arg(from_time)
  AND visit.scheduled_time < sqlc.arg(to_time)
ORDER BY visit.scheduled_time;

-- name: RecordVisitEta :one
-- 위치를 받을 때마다 ETA를 갱신하고, 예정 상태면 이동중으로 바꿉니다.
-- 끝난 방문(완료·취소)은 건드리지 않으므로 행이 없으면 없는 방문이거나 끝난 방문입니다.
UPDATE visit
SET eta_current = sqlc.arg(eta_current),
    status = CASE WHEN status = 'SCHEDULED' THEN 'EN_ROUTE' ELSE status END
WHERE id = sqlc.arg(id)
  AND status IN ('SCHEDULED', 'EN_ROUTE', 'SESSION_ACTIVE')
RETURNING *;

-- name: ClaimVisitSession :one
-- 세션 시작 권한을 한 문장으로 차지합니다. 위치가 동시에 두 번 와도 한 요청만 행을 돌려받고,
-- 나머지는 pgx.ErrNoRows를 받습니다. 같은 트랜잭션에서 CreatePickupSession을 이어서 호출합니다.
UPDATE visit
SET status = 'SESSION_ACTIVE'
WHERE id = $1
  AND status IN ('SCHEDULED', 'EN_ROUTE')
RETURNING *;

-- name: CompleteVisit :one
-- 조무사 도착. 이미 완료·취소된 방문이면 행이 없습니다.
UPDATE visit
SET status = 'COMPLETED',
    actual_arrival_time = sqlc.arg(arrived_at)
WHERE id = sqlc.arg(id)
  AND status IN ('SCHEDULED', 'EN_ROUTE', 'SESSION_ACTIVE')
RETURNING *;
