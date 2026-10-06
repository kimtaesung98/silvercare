-- +goose Up
-- 긴 말동무 대화의 앞부분 요약. 대화 요청에는 이 요약과 그 뒤 발화만 넣어
-- 입력 토큰이 대화 길이만큼 늘지 않게 합니다. history_summary_through는 요약에
-- 들어간 마지막 발화의 seq입니다.
ALTER TABLE conversation_session
    ADD COLUMN history_summary         text,
    ADD COLUMN history_summary_through integer,
    ADD CONSTRAINT conversation_session_history_summary_pair
        CHECK ((history_summary IS NULL) = (history_summary_through IS NULL));

-- +goose Down
ALTER TABLE conversation_session
    DROP CONSTRAINT conversation_session_history_summary_pair,
    DROP COLUMN history_summary_through,
    DROP COLUMN history_summary;
