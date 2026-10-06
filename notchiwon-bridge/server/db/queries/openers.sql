-- name: ListActiveOpenerClips :many
-- 태블릿이 내려받을 0번 문장 목록. 어르신의 preferred_tts_voice로 거릅니다.
SELECT * FROM opener_clip
WHERE voice = $1
  AND active
ORDER BY category, id;

-- name: GetOpenerClip :one
SELECT * FROM opener_clip WHERE id = $1;
