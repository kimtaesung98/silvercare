-- name: GetBriefingBySession :one
SELECT * FROM briefing_report WHERE session_id = $1;

-- name: GetBriefingDetail :one
-- 브리핑 카드용: 세션의 감정 태그와 위급 횟수를 함께 읽습니다.
SELECT
    sqlc.embed(briefing_report),
    conversation_session.overall_emotion_tag,
    (SELECT count(*) FROM escalation_event WHERE escalation_event.session_id = briefing_report.session_id)::integer
        AS escalation_count
FROM briefing_report
JOIN conversation_session ON conversation_session.id = briefing_report.session_id
WHERE briefing_report.session_id = $1;

-- name: UpsertBriefing :one
-- 작업이 다시 돌아도(재시도) 브리핑은 세션당 하나입니다.
INSERT INTO briefing_report (session_id, summary_text, top_keywords, emotion_flag, model)
VALUES (sqlc.arg(session_id), sqlc.arg(summary_text), sqlc.arg(top_keywords), sqlc.narg(emotion_flag), sqlc.narg(model))
ON CONFLICT (session_id) DO UPDATE
SET summary_text = EXCLUDED.summary_text,
    top_keywords = EXCLUDED.top_keywords,
    emotion_flag = EXCLUDED.emotion_flag,
    model = EXCLUDED.model,
    generated_at = now()
RETURNING *;

-- name: MarkBriefingRead :one
UPDATE briefing_report
SET read_by_caregiver_at = COALESCE(read_by_caregiver_at, now())
WHERE session_id = $1
RETURNING *;
