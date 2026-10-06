-- +goose Up
-- 어르신 댁 위치. 실제 ETA(카카오모빌리티 길찾기)의 목적지이고, 조무사 앱 지도에
-- 표시합니다. 비어 있으면 서버는 예정 시각 기준 ETA로 대신합니다.
ALTER TABLE elder
    ADD COLUMN home_address   text,
    ADD COLUMN home_latitude  double precision CHECK (home_latitude BETWEEN -90 AND 90),
    ADD COLUMN home_longitude double precision CHECK (home_longitude BETWEEN -180 AND 180),
    ADD CONSTRAINT elder_home_coordinates_pair
        CHECK ((home_latitude IS NULL) = (home_longitude IS NULL));

-- +goose Down
ALTER TABLE elder
    DROP CONSTRAINT elder_home_coordinates_pair,
    DROP COLUMN home_longitude,
    DROP COLUMN home_latitude,
    DROP COLUMN home_address;
