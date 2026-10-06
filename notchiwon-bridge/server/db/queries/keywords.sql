-- name: ListTopKeywords :many
SELECT * FROM keyword_tag
WHERE elder_id = sqlc.arg(elder_id)
ORDER BY score DESC
LIMIT sqlc.arg(max_count);

-- name: BumpKeyword :one
-- 시간 감쇠 갱신: score = score * 0.9 + 이번 세션 언급 수.
INSERT INTO keyword_tag (elder_id, keyword, category, score, emotion_tone, last_mentioned_at, mention_count_total)
VALUES (
    sqlc.arg(elder_id), sqlc.arg(keyword), sqlc.arg(category), sqlc.arg(mentions)::integer,
    sqlc.arg(emotion_tone), sqlc.arg(mentioned_at), sqlc.arg(mentions)::integer
)
ON CONFLICT (elder_id, keyword) DO UPDATE
SET score = keyword_tag.score * 0.9 + EXCLUDED.score,
    category = EXCLUDED.category,
    emotion_tone = EXCLUDED.emotion_tone,
    last_mentioned_at = EXCLUDED.last_mentioned_at,
    mention_count_total = keyword_tag.mention_count_total + EXCLUDED.mention_count_total
RETURNING *;
