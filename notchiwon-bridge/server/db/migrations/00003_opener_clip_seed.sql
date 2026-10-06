-- 단계 3: 0번 문장 기본 목록 (architecture.md 6.2절).
-- 기쁨과 슬픔 모두에 어울리는 중립적인 맞장구만 둡니다. 음성은 단계 4에서 Clova TTS로 합성해
-- audio_ref를 채우므로, 그 전까지는 'pending:tts'이고 duration_ms는 글자 수로 어림한 값입니다.
-- ESCALATION 문장은 docs/specs/conversation-loop.md 2절의 안심 문장과 같아야 합니다.

-- +goose Up
INSERT INTO opener_clip (category, text, voice, audio_ref, duration_ms) VALUES
    ('RECALL',       '아이고, 그러셨어요.',                                         'default', 'pending:tts', 1400),
    ('RECALL',       '아, 그러셨구나.',                                             'default', 'pending:tts', 1200),
    ('RECALL',       '음, 그랬군요.',                                               'default', 'pending:tts', 1100),
    ('QUESTION',     '음, 그건 말이죠.',                                            'default', 'pending:tts', 1300),
    ('QUESTION',     '아, 그거요.',                                                 'default', 'pending:tts', 1000),
    ('QUESTION',     '어디 보자.',                                                  'default', 'pending:tts', 1000),
    ('EMOTION',      '그러셨구나, 마음이 그러셨어요.',                              'default', 'pending:tts', 2000),
    ('EMOTION',      '아이고, 그런 마음이 드셨어요.',                               'default', 'pending:tts', 1900),
    ('SHORT_ANSWER', '네, 네.',                                                     'default', 'pending:tts', 800),
    ('SHORT_ANSWER', '그럼요.',                                                     'default', 'pending:tts', 800),
    ('SHORT_ANSWER', '그렇죠.',                                                     'default', 'pending:tts', 800),
    ('GREETING',     '네, 안녕하세요.',                                             'default', 'pending:tts', 1200),
    ('GREETING',     '네, 반가워요.',                                               'default', 'pending:tts', 1100),
    ('ESCALATION',   '지금 많이 힘드시죠. 선생님께 바로 알려드렸어요. 조금만 기다려주세요.', 'default', 'pending:tts', 4500),
    ('FILLER',       '음...',                                                       'default', 'pending:tts', 700),
    ('FILLER',       '그러니까요...',                                               'default', 'pending:tts', 1000),
    ('FILLER',       '어디 보자...',                                                'default', 'pending:tts', 1000);

-- +goose Down
DELETE FROM opener_clip
WHERE voice = 'default'
  AND audio_ref = 'pending:tts'
  AND id NOT IN (SELECT opener_clip_id FROM utterance WHERE opener_clip_id IS NOT NULL);
