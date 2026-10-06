-- 단계 2: 조무사 임시 로그인(센터 발급 계정)과 세션 선점 종료 사유.
-- 로그인 방식(전화번호 인증 vs 센터 발급 계정)이 정해지면 새 마이그레이션으로 바꿉니다.

-- +goose Up
ALTER TABLE caregiver
    ADD COLUMN login_id      text UNIQUE,
    ADD COLUMN password_hash text,
    ADD CONSTRAINT caregiver_login_pair CHECK ((login_id IS NULL) = (password_hash IS NULL));

-- 픽업 대기 세션이 시작될 때 진행 중이던 말동무 세션은 PREEMPTED로 끝납니다.
ALTER TABLE conversation_session DROP CONSTRAINT conversation_session_ended_reason_check;
ALTER TABLE conversation_session ADD CONSTRAINT conversation_session_ended_reason_check
    CHECK (ended_reason IN (
        'CAREGIVER_ARRIVED', 'ELDER_DECLINED', 'TIMEOUT', 'ERROR',
        'NO_RESPONSE', 'BEDTIME', 'TOKEN_LIMIT', 'PREEMPTED'));

-- +goose Down
UPDATE conversation_session SET ended_reason = 'ERROR' WHERE ended_reason = 'PREEMPTED';
ALTER TABLE conversation_session DROP CONSTRAINT conversation_session_ended_reason_check;
ALTER TABLE conversation_session ADD CONSTRAINT conversation_session_ended_reason_check
    CHECK (ended_reason IN (
        'CAREGIVER_ARRIVED', 'ELDER_DECLINED', 'TIMEOUT', 'ERROR',
        'NO_RESPONSE', 'BEDTIME', 'TOKEN_LIMIT'));

ALTER TABLE caregiver
    DROP CONSTRAINT caregiver_login_pair,
    DROP COLUMN password_hash,
    DROP COLUMN login_id;
