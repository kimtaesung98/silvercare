-- +goose Up
-- 보호자 앱 로그인(센터 발급 계정). 조무사와 같은 방식이고, 로그인 방식이
-- 정해지면 새 마이그레이션으로 바꿉니다(결정 14).
ALTER TABLE guardian
    ADD COLUMN login_id      text UNIQUE,
    ADD COLUMN password_hash text,
    ADD CONSTRAINT guardian_login_pair CHECK ((login_id IS NULL) = (password_hash IS NULL));

-- 보호자가 확인한 위급. 조무사와 보호자 중 먼저 누른 쪽이 남습니다.
ALTER TABLE escalation_event
    ADD COLUMN acknowledged_by_guardian_id uuid REFERENCES guardian (id);

-- +goose Down
ALTER TABLE escalation_event
    DROP COLUMN acknowledged_by_guardian_id;

ALTER TABLE guardian
    DROP CONSTRAINT guardian_login_pair,
    DROP COLUMN password_hash,
    DROP COLUMN login_id;
