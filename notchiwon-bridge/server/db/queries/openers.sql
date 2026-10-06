-- name: ListActiveOpenerClips :many
-- 태블릿이 내려받을 0번 문장 목록. 어르신의 preferred_tts_voice로 거릅니다.
SELECT * FROM opener_clip
WHERE voice = $1
  AND active
ORDER BY category, id;

-- name: GetOpenerClip :one
SELECT * FROM opener_clip WHERE id = $1;

-- name: GetOpenerClipAudio :one
SELECT mp3 FROM opener_clip_audio WHERE clip_id = $1;

-- name: ListOpenerClipsWithoutAudio :many
-- cmd/admin synth-openers가 합성할 클립.
SELECT c.* FROM opener_clip c
LEFT JOIN opener_clip_audio a ON a.clip_id = c.id
WHERE c.active AND a.clip_id IS NULL
ORDER BY c.voice, c.category, c.id;

-- name: SaveOpenerClipAudio :exec
WITH saved AS (
    INSERT INTO opener_clip_audio (clip_id, mp3) VALUES (sqlc.arg(clip_id), sqlc.arg(mp3))
    ON CONFLICT (clip_id) DO UPDATE SET mp3 = EXCLUDED.mp3, created_at = now()
    RETURNING clip_id
)
UPDATE opener_clip SET audio_ref = 'db' WHERE id IN (SELECT clip_id FROM saved);
