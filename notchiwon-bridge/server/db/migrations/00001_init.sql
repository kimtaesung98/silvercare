-- 초기 스키마. backend/prisma/schema.prisma의 테이블 10개를 이름 그대로 옮기고
-- docs/architecture.md 5절의 변경(말동무 세션, 0번 문장, 기기 인증 등)을 반영합니다.
-- 이미 머지된 마이그레이션은 고치지 않고 새 파일을 추가합니다.

-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TABLE guardian (
    id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    text NOT NULL,
    phone                   text NOT NULL,
    notification_preference jsonb,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE daycare_center (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    address    text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE elder (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name                text NOT NULL,
    birth_date          date NOT NULL,
    dementia_stage      text NOT NULL CHECK (dementia_stage IN ('MILD', 'MODERATE', 'SEVERE')),
    preferred_tts_voice text,
    guardian_id         uuid NOT NULL REFERENCES guardian (id),
    center_id           uuid NOT NULL REFERENCES daycare_center (id),
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX elder_guardian_id_idx ON elder (guardian_id);
CREATE INDEX elder_center_id_idx ON elder (center_id);

CREATE TABLE caregiver (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    center_id  uuid NOT NULL REFERENCES daycare_center (id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX caregiver_center_id_idx ON caregiver (center_id);

CREATE TABLE visit (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    elder_id            uuid NOT NULL REFERENCES elder (id),
    caregiver_id        uuid NOT NULL REFERENCES caregiver (id),
    scheduled_time      timestamptz NOT NULL,
    eta_current         timestamptz,
    status              text NOT NULL DEFAULT 'SCHEDULED'
                        CHECK (status IN ('SCHEDULED', 'EN_ROUTE', 'SESSION_ACTIVE', 'COMPLETED', 'CANCELLED')),
    actual_arrival_time timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX visit_elder_id_scheduled_time_idx ON visit (elder_id, scheduled_time);
CREATE INDEX visit_caregiver_id_scheduled_time_idx ON visit (caregiver_id, scheduled_time);

CREATE TABLE conversation_session (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- 말동무 세션은 방문과 무관하므로 NULL. 방문 하나에 세션은 최대 하나.
    visit_id            uuid UNIQUE REFERENCES visit (id),
    elder_id            uuid NOT NULL REFERENCES elder (id),
    mode                text NOT NULL CHECK (mode IN ('PICKUP_BRIDGE', 'COMPANION')),
    started_by          text NOT NULL CHECK (started_by IN ('SYSTEM', 'ELDER', 'SCHEDULE')),
    started_at          timestamptz NOT NULL DEFAULT now(),
    ended_at            timestamptz,
    trigger_eta_minutes integer,
    overall_emotion_tag text CHECK (overall_emotion_tag IN ('STABLE', 'SLIGHTLY_ANXIOUS', 'UNUSUAL')),
    ended_reason        text CHECK (ended_reason IN (
                            'CAREGIVER_ARRIVED', 'ELDER_DECLINED', 'TIMEOUT', 'ERROR',
                            'NO_RESPONSE', 'BEDTIME', 'TOKEN_LIMIT')),
    prompt_version      text,
    CHECK ((mode = 'PICKUP_BRIDGE') = (visit_id IS NOT NULL)),
    CHECK (mode = 'COMPANION' OR trigger_eta_minutes IS NOT NULL),
    CHECK ((ended_at IS NULL) = (ended_reason IS NULL))
);

-- 어르신 한 명에게 진행 중인 세션은 하나뿐. 위치가 동시에 두 번 와도 두 번째 INSERT는 실패합니다.
CREATE UNIQUE INDEX conversation_session_one_open_per_elder_idx
    ON conversation_session (elder_id) WHERE ended_at IS NULL;
CREATE INDEX conversation_session_elder_id_started_at_idx ON conversation_session (elder_id, started_at);

CREATE TABLE opener_clip (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    category    text NOT NULL CHECK (category IN (
                    'RECALL', 'QUESTION', 'EMOTION', 'SHORT_ANSWER', 'GREETING', 'ESCALATION', 'FILLER')),
    text        text NOT NULL,
    voice       text NOT NULL,
    audio_ref   text NOT NULL,
    duration_ms integer NOT NULL CHECK (duration_ms > 0),
    active      boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (category, text, voice)
);

CREATE TABLE utterance (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     uuid NOT NULL REFERENCES conversation_session (id),
    -- seq: 세션 안의 턴 번호. chunk_index: 턴 안의 문장 번호(AI의 0번 문장 = 0, 스트리밍 문장 = 1..n).
    seq            integer NOT NULL CHECK (seq >= 0),
    chunk_index    integer NOT NULL DEFAULT 0 CHECK (chunk_index >= 0),
    speaker        text NOT NULL CHECK (speaker IN ('ELDER', 'AI')),
    text           text NOT NULL,
    audio_ref      text,
    emotion_score  double precision,
    flagged_risk   boolean NOT NULL DEFAULT false,
    opener_clip_id uuid REFERENCES opener_clip (id),
    model          text,
    latency_ms     integer,
    input_tokens   integer,
    output_tokens  integer,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (session_id, seq, chunk_index),
    CHECK (opener_clip_id IS NULL OR (speaker = 'AI' AND chunk_index = 0))
);

CREATE INDEX utterance_session_id_created_at_idx ON utterance (session_id, created_at);

CREATE TABLE keyword_tag (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    elder_id            uuid NOT NULL REFERENCES elder (id),
    keyword             text NOT NULL,
    category            text NOT NULL CHECK (category IN ('FAMILY', 'HOBBY', 'MEMORY', 'FOOD', 'HEALTH_OTHER')),
    score               double precision NOT NULL DEFAULT 0,
    emotion_tone        text NOT NULL DEFAULT 'NEUTRAL' CHECK (emotion_tone IN ('POSITIVE', 'NEUTRAL', 'AVOIDANT')),
    last_mentioned_at   timestamptz NOT NULL,
    mention_count_total integer NOT NULL DEFAULT 1,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (elder_id, keyword)
);

CREATE INDEX keyword_tag_elder_id_score_idx ON keyword_tag (elder_id, score DESC);

CREATE TABLE briefing_report (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id           uuid NOT NULL UNIQUE REFERENCES conversation_session (id),
    summary_text         text NOT NULL,
    top_keywords         jsonb NOT NULL,
    emotion_flag         text,
    model                text,
    generated_at         timestamptz NOT NULL DEFAULT now(),
    read_by_caregiver_at timestamptz
);

CREATE TABLE escalation_event (
    id                          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id                  uuid NOT NULL REFERENCES conversation_session (id),
    utterance_id                uuid REFERENCES utterance (id),
    trigger_type                text NOT NULL CHECK (trigger_type IN (
                                    'FALL_MENTION', 'PAIN_COMPLAINT', 'SELF_OR_OTHER_HARM', 'OTHER_ANOMALY')),
    -- RULE: 규칙 필터(rule_id 필수), LLM: Claude의 flag_concern 도구(reason에 설명).
    source                      text NOT NULL CHECK (source IN ('RULE', 'LLM')),
    rule_id                     text,
    reason                      text,
    notified_targets            jsonb NOT NULL,
    created_at                  timestamptz NOT NULL DEFAULT now(),
    acknowledged_at             timestamptz,
    acknowledged_by_caregiver_id uuid REFERENCES caregiver (id),
    resolved_at                 timestamptz,
    CHECK ((source = 'RULE') = (rule_id IS NOT NULL))
);

CREATE INDEX escalation_event_session_id_idx ON escalation_event (session_id);
CREATE INDEX escalation_event_open_idx ON escalation_event (created_at) WHERE acknowledged_at IS NULL;

-- 태블릿·휴대폰 기기. 태블릿은 token_hash로 인증하고, 휴대폰은 FCM 토큰만 등록합니다.
CREATE TABLE device (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind         text NOT NULL CHECK (kind IN ('ELDER_TABLET', 'CAREGIVER_PHONE', 'GUARDIAN_PHONE')),
    elder_id     uuid REFERENCES elder (id),
    caregiver_id uuid REFERENCES caregiver (id),
    guardian_id  uuid REFERENCES guardian (id),
    label        text,
    -- 기기 토큰의 SHA-256. 원문 토큰은 저장하지 않습니다.
    token_hash   bytea UNIQUE,
    fcm_token    text UNIQUE,
    last_seen_at timestamptz,
    revoked_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (kind = 'ELDER_TABLET'    AND elder_id IS NOT NULL AND caregiver_id IS NULL AND guardian_id IS NULL AND token_hash IS NOT NULL) OR
        (kind = 'CAREGIVER_PHONE' AND caregiver_id IS NOT NULL AND elder_id IS NULL AND guardian_id IS NULL) OR
        (kind = 'GUARDIAN_PHONE'  AND guardian_id IS NOT NULL AND elder_id IS NULL AND caregiver_id IS NULL)
    )
);

CREATE INDEX device_elder_id_idx ON device (elder_id) WHERE revoked_at IS NULL;
CREATE INDEX device_caregiver_id_idx ON device (caregiver_id) WHERE revoked_at IS NULL;
CREATE INDEX device_guardian_id_idx ON device (guardian_id) WHERE revoked_at IS NULL;

-- 조무사 위치. ETA 트리거 검증용이며 짧게 보관합니다(received_at 기준 삭제).
CREATE TABLE caregiver_location (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    caregiver_id uuid NOT NULL REFERENCES caregiver (id),
    visit_id     uuid REFERENCES visit (id),
    latitude     double precision NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude    double precision NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    accuracy_m   double precision,
    eta_minutes  integer,
    recorded_at  timestamptz NOT NULL,
    received_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX caregiver_location_visit_id_received_at_idx ON caregiver_location (visit_id, received_at DESC);
CREATE INDEX caregiver_location_received_at_idx ON caregiver_location (received_at);

-- 말동무 모드 설정. 어르신당 한 행.
CREATE TABLE companion_schedule (
    elder_id          uuid PRIMARY KEY REFERENCES elder (id),
    enabled           boolean NOT NULL DEFAULT true,
    -- 보호자가 정한 안부 대화 시각(현지 시각).
    check_in_times    time[] NOT NULL DEFAULT '{}',
    -- 취침 시간에는 말동무를 시작하지 않고 진행 중이면 마무리합니다. 자정을 넘어갈 수 있습니다.
    bedtime_start     time,
    bedtime_end       time,
    time_zone         text NOT NULL DEFAULT 'Asia/Seoul',
    daily_token_limit integer CHECK (daily_token_limit > 0),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CHECK ((bedtime_start IS NULL) = (bedtime_end IS NULL))
);

-- 보호자용 하루 요약.
CREATE TABLE daily_digest (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    elder_id         uuid NOT NULL REFERENCES elder (id),
    digest_date      date NOT NULL,
    summary_text     text NOT NULL,
    emotion_flag     text,
    session_count    integer NOT NULL DEFAULT 0,
    escalation_count integer NOT NULL DEFAULT 0,
    total_tokens     integer NOT NULL DEFAULT 0,
    model            text,
    generated_at     timestamptz NOT NULL DEFAULT now(),
    sent_at          timestamptz,
    UNIQUE (elder_id, digest_date)
);

CREATE TRIGGER guardian_set_updated_at BEFORE UPDATE ON guardian
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER elder_set_updated_at BEFORE UPDATE ON elder
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER visit_set_updated_at BEFORE UPDATE ON visit
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER keyword_tag_set_updated_at BEFORE UPDATE ON keyword_tag
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER companion_schedule_set_updated_at BEFORE UPDATE ON companion_schedule
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE daily_digest;
DROP TABLE companion_schedule;
DROP TABLE caregiver_location;
DROP TABLE device;
DROP TABLE escalation_event;
DROP TABLE briefing_report;
DROP TABLE keyword_tag;
DROP TABLE utterance;
DROP TABLE opener_clip;
DROP TABLE conversation_session;
DROP TABLE visit;
DROP TABLE caregiver;
DROP TABLE elder;
DROP TABLE daycare_center;
DROP TABLE guardian;
DROP FUNCTION set_updated_at();
