-- name: CreateUtterance :one
INSERT INTO utterance (
    session_id, seq, chunk_index, speaker, text, audio_ref, emotion_score, flagged_risk,
    opener_clip_id, model, latency_ms, input_tokens, output_tokens
) VALUES (
    sqlc.arg(session_id), sqlc.arg(seq), sqlc.arg(chunk_index), sqlc.arg(speaker), sqlc.arg(text),
    sqlc.narg(audio_ref), sqlc.narg(emotion_score), sqlc.arg(flagged_risk),
    sqlc.narg(opener_clip_id), sqlc.narg(model), sqlc.narg(latency_ms),
    sqlc.narg(input_tokens), sqlc.narg(output_tokens)
)
RETURNING *;

-- name: NextUtteranceSeq :one
SELECT COALESCE(MAX(seq) + 1, 0)::integer AS next_seq FROM utterance WHERE session_id = $1;

-- name: ListSessionUtterances :many
SELECT * FROM utterance WHERE session_id = $1 ORDER BY seq, chunk_index;

-- name: SumElderTokensSince :one
-- 말동무 하루 토큰 한도 확인용.
SELECT COALESCE(SUM(COALESCE(u.input_tokens, 0) + COALESCE(u.output_tokens, 0)), 0)::bigint AS total_tokens
FROM utterance u
JOIN conversation_session s ON s.id = u.session_id
WHERE s.elder_id = sqlc.arg(elder_id)
  AND u.created_at >= sqlc.arg(since);
