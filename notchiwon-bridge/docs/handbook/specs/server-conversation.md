# 명세: 대화 엔진 (session · opener · llm · speech)

## 목적

어르신 발화 하나를 AI 턴 하나로 바꿉니다. 0번 문장(미리 녹음한 맞장구)을 바로 틀고, 그동안 Claude가 쓰는 1번부터의 문장을 문장 단위로 태블릿에 흘려보냅니다. 원래 규칙은 [conversation-loop.md](../../specs/conversation-loop.md)입니다.

## 경계

- 한다: 턴 직렬화와 취소(끼어들기), 규칙 필터 호출, 0번 문장 선택, Claude 스트리밍과 문장 분할, 추임새·시간 제한·대체 문장, 말동무 작별, 긴 대화 요약 요청, `/ws/elder` 연결 관리, 서버 STT·TTS.
- 하지 않는다: 위급 규칙 자체와 알림(→ [server-safety.md](./server-safety.md)), 세션 시작 조건(픽업 → [server-pickup.md](./server-pickup.md), 말동무 → [server-companion.md](./server-companion.md)), 브리핑(→ server-pickup).

## 파일

| 파일 | 내용 |
| --- | --- |
| `server/internal/session/engine.go` | `Engine`: `HandleElderText`, `Greet`, `BargeIn`, `Forget`, 턴 생성 `generate` |
| `server/internal/session/ws.go` | `Hub`(연결 등록, 세션 알림 받기), `conn`(읽기·쓰기·이벤트 고루틴), `wsSink`(턴 이벤트 → 프레임, TTS) |
| `server/internal/session/split.go` | 스트림을 문장으로 자르는 `splitter` |
| `server/internal/session/goodbye.go` | 말동무 종료 문구 `WantsToStop` |
| `server/internal/opener/` | `Classify`(규칙 분류), `Library.Manifest`/`Pick`, 0번 문장 음성 합성·보관 |
| `server/internal/llm/` | `Client` 인터페이스, `Anthropic`(스트리밍, `flag_concern` 도구), 프롬프트 템플릿(`prompts/*.tmpl`), 요약기(브리핑·하루 소식·긴 대화) |
| `server/internal/speech/` | `Recognizer`/`Synthesizer`/`Unavailable`(`speech.go`), `Transcribe`(`transcribe.go`, 스트리밍 인식), `Polly`(`polly.go`), `New`(`aws.go`, 자격 증명이 있으면 둘 다) |
| 테스트 | `session/engine_test.go`, `ws_test.go`, `split_test.go`, `companion_test.go`, `opener/classify_test.go`, `llm/*_test.go`, `speech/speech_test.go` |

## 인터페이스

| 방향 | 이름 | 설명 |
| --- | --- | --- |
| 제공 | `Engine.HandleElderText(ctx, sessionID, text, sink) (TurnResult, error)` | 한 턴. 오류는 `ErrEmptyText`, `ErrSessionNotFound`, `ErrSessionEnded`, 그 밖엔 DB 오류. Claude 실패는 오류가 아님 |
| 제공 | `Engine.Greet`, `BargeIn`, `Forget` | 0번째 턴 인사, 진행 중 턴 취소, 끝난 세션 정리 |
| 제공 | `Hub` (`http.Handler`) | `GET /ws/elder`. `visit.SessionNotifier`, `companion.Tablets`를 구현 |
| 제공 | `Hub.End(ctx, sessionID, reason)` | 세션을 끝내고 `EndListener`들(자기 자신, 브리핑 큐)에 알림 |
| 기대 | `Sink` | 턴 이벤트를 순서대로 받음 (`wsSink`가 구현) |
| 기대 | `Alerter`, `Budget`, `Compactor` | 위급 알림 큐, 말동무 토큰 한도, 긴 대화 요약 큐 (`main.go`에서 `Set*`) |
| 기대 | `llm.Client`, `speech.Recognizer`, `speech.Synthesizer` | 키가 없으면 `Unavailable` 구현이 들어감 |

## 요구사항

- `CONV-01`: 한 세션의 턴은 한 번에 하나씩 돕니다. 새 발화나 끼어들기는 진행 중 턴의 컨텍스트를 취소합니다 (`sessionState.begin`).
- `CONV-02`: 어르신 발화는 턴이 취소돼도 저장합니다 (쓰기는 `context.WithoutCancel`).
- `CONV-03`: 규칙 필터(`escalation.Check`)가 Claude보다 먼저 돕니다. 걸리면 위급 이벤트를 저장하고 `ESCALATION` 0번 문장(없으면 `ReassuranceSentence`)만 말하며 Claude를 부르지 않습니다.
- `CONV-04`: 말동무 모드에서 규칙 필터를 통과한 뒤 `WantsToStop`이면 `GoodbyeSentence`로, 토큰 한도를 넘었으면 `TokenLimitSentence`로 답하고 세션을 끝냅니다 (`TurnResult.End`). 순서 때문에 "죽고 싶어, 그만할래"는 위급으로 처리됩니다.
- `CONV-05`: 0번 문장은 `opener.Classify`(정규식, LLM 아님)로 고른 분류에서 무작위로 고르되 직전 클립은 피합니다.
- `CONV-06`: Claude 요청은 시스템 블록 [모드 블록(캐시), 어르신 이름·상위 관심사 5개, (있으면) 긴 대화 요약] + 시작 지시 + 저장된 대화입니다. 0번 문장은 `Continuation`으로 "이미 말했다"고 알려 1번부터 쓰게 합니다.
- `CONV-07`: 스트림은 마침표·물음표·느낌표·줄바꿈 뒤 공백에서 문장으로 자르고, 문장마다 저장 → `ai.reply`(TTS 있으면 MP3 함께)로 보냅니다. 한 턴은 최대 4문장입니다.
- `CONV-08`: 1번 문장이 2초 안에 없으면 `FILLER` 클립 하나를 보내고, 6초 안에 없거나 Claude가 실패·거절하면 대체 문장(픽업 `FallbackSentence`, 말동무 `CompanionFallbackSentence`)을 말합니다. 턴 전체는 20초에서 자릅니다.
- `CONV-09`: Claude의 `flag_concern` 도구 호출은 `source=LLM` 위급 이벤트가 됩니다. 입력을 못 읽어도 `OTHER_ANOMALY`로 남깁니다.
- `CONV-10`: 마지막 AI 문장에 모델과 입출력 토큰을 기록합니다 (말동무 하루 한도 계산의 근거).
- `CONV-11`: 요약되지 않은 발화가 30개를 넘으면 마지막 20개를 뺀 앞부분 요약을 큐에 넣고, 요약이 저장되면 그 부분은 요청에서 빠집니다.
- `CONV-12`: 세션의 첫 턴(seq 0)은 서버가 먼저 인사합니다 (`Greet`). 태블릿이 재접속했을 때 아직 대화가 없으면 그때 인사합니다.
- `CONV-13`: 태블릿 연결은 기기당 하나입니다. 새 연결이 오면 옛 연결을 `StatusPolicyViolation`으로 닫습니다.
- `CONV-14`: 한 턴의 이벤트는 연결의 이벤트 고루틴에서 순서대로 나갑니다. TTS도 거기서 하므로 문장 순서가 보장됩니다. 문장 TTS는 5초 제한, 넘으면 글자만 보냅니다.
- `CONV-15`: 보내기 큐(256)가 차면 느린 태블릿으로 보고 연결을 닫습니다. 태블릿은 재접속해 `connection.ready`부터 이어갑니다.
- `CONV-16`: `elder.audio`는 STT(10초 제한)를 거쳐 `elder.text`와 같은 턴이 됩니다. Transcribe 스트리밍에 녹음을 100ms 조각으로 보내고 입력을 닫은 뒤 확정(non-partial) 결과만 이어 붙입니다. `pcm16`, `opus`(Ogg)만 받고, 인식 실패·빈 결과·`aac`는 `STT_FAILED`.
- `CONV-17`: Claude 키와 AWS 자격 증명은 서버에만 있습니다. 태블릿은 녹음하고 틀기만 합니다 (기기 음성 모드는 기기 엔진이 인식하고 글자만 보냄, [apps.md](./apps.md)).

## 설정값

| 환경변수 | 기본 | 뜻 |
| --- | --- | --- |
| `ANTHROPIC_API_KEY` | 없음 | 없으면 매 턴 대체 문장 |
| `CLAUDE_CONVERSATION_MODEL` | `claude-sonnet-5-5` | 대화 모델. 이 모델이면 effort low, thinking `between_tools` |
| `CLAUDE_SUMMARY_MODEL` | `claude-haiku-4-5` | 브리핑·하루 소식·긴 대화 요약 |
| `CONVERSATION_FILLER_AFTER` | `2s` | 추임새 시각 |
| `CONVERSATION_FIRST_SENTENCE_TIMEOUT` | `6s` | 대체 문장 시각 |
| `CONVERSATION_TURN_TIMEOUT` | `20s` | 턴 상한 (첫 문장 제한 이상이어야 함) |
| `AWS_PROFILE` 또는 `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` | 없음 | 없으면 서버 STT 실패(태블릿 기기 인식은 됨), TTS 없음 |
| `AWS_REGION`, `TRANSCRIBE_LANGUAGE` | `ap-southeast-2`, `ko-KR` | Transcribe·Polly 리전, 인식 언어 |
| `POLLY_VOICE`, `POLLY_ENGINE`, `POLLY_RATE` | `Seoyeon`, `neural`, `90%` | Polly 목소리(Seoyeon·Jihye), 엔진, SSML 빠르기. 어르신 `preferred_tts_voice`가 Seoyeon·Jihye면 그 목소리 |

## 실패할 때

| 상황 | 동작 |
| --- | --- |
| Claude 실패·거절·늦음 | 대체 문장, `outcome=FALLBACK`, 오류 로그 |
| 0번 문장 없음 | 0번 없이 1번부터 (경고 로그) |
| 위급 이벤트 저장 실패 | 어르신은 안심 문장을 듣고, 오류 로그 (알림 없음) |
| TTS 실패 | 글자만 보냄 (태블릿이 기기 TTS로 읽음) |
| STT 실패 | `STT_FAILED`, 태블릿이 "다시 말씀해 주세요" |
| DB 실패 | `error {code: INTERNAL}` |

## 확장 지점

- 다른 LLM: `llm.Client` 하나(`Stream`)를 구현해 `main.go`에서 바꿉니다. 도구 호출은 `Result.Concerns`로 돌려줍니다.
- 다른 STT·TTS: `speech.Recognizer`/`Synthesizer` 구현. `Synthesize`는 MP3를 돌려줘야 태블릿 재생기와 맞습니다.
- 프롬프트: `llm/prompts/*.tmpl`. 고치면 `PromptVersion`(파일 해시)이 바뀌어 새 세션에 기록됩니다.
- 0번 문장: `opener_clip` 행(마이그레이션 `00003` 시드). 문구를 바꾸면 목록 해시가 바뀌어 태블릿이 다시 받습니다.

## 알려진 문제

- [CR-02](../code-review.md#cr-02): 잘린 답·시간 초과에서 `flag_concern`과 토큰 사용량이 버려짐 (CONV-09, CONV-10 위반).
- [CR-03](../code-review.md#cr-03): 긴 대화 요약이 세션당 한 번만 들어감 (CONV-11 위반).
- [CR-06](../code-review.md#cr-06), [CR-07](../code-review.md#cr-07): 끼어들기 기록과 상태 정리.
- [CR-15](../code-review.md#cr-15): 프롬프트에 들어가는 관심사 키워드가 시간이 지나도 줄지 않음.

## 다시 만들 때

- 지켜야 할 것: 규칙 필터가 먼저(CONV-03), 어르신 말은 항상 저장(CONV-02), 한 세션 한 턴(CONV-01), 프레임 짝(CTR-04).
- 함정: 소켓 읽기·쓰기에 연결 컨텍스트를 넘기면 취소할 때 닫힘 코드 없이 끊깁니다. 지금 코드는 읽기·쓰기에 따로 타임아웃 컨텍스트를 씁니다 (`conn.close` 주석).
- 함정: 턴 컨텍스트(`tctx`)와 쓰기 컨텍스트(`wctx`)를 섞으면 취소된 턴의 기록이 빠집니다.
