-- 단계 4: Clova TTS로 합성한 0번 문장 음성(MP3). 클립 하나에 20KB 안팎이라 DB에 둡니다.
-- 합성되면 opener_clip.audio_ref가 'db'가 되어 목록 version이 바뀌고, 태블릿이 음성을 내려받습니다.

-- +goose Up
CREATE TABLE opener_clip_audio (
    clip_id    uuid PRIMARY KEY REFERENCES opener_clip (id) ON DELETE CASCADE,
    mp3        bytea NOT NULL CHECK (length(mp3) > 0),
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
UPDATE opener_clip SET audio_ref = 'pending:tts' WHERE audio_ref = 'db';
DROP TABLE opener_clip_audio;
