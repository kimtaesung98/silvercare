# `/ws/elder` WebSocket 이벤트

> 기준일: 2026-10-06. 어르신 태블릿 앱과 서버 사이의 실시간 계약입니다. REST는 [openapi.yaml](./openapi.yaml)을 보세요.
> 메시지 형태의 기준은 [ws-events.schema.json](./ws-events.schema.json)이고, 이 문서는 흐름과 규칙을 설명합니다.
> 모든 이벤트의 예시는 [ws-events.examples.json](./ws-events.examples.json)에 있으며, `go test ./api`가 예시를 스키마로 검사합니다.

## 1. 연결

- 주소: `GET /ws/elder` (WebSocket 업그레이드)
- 인증: `Authorization: Bearer <기기 토큰>` 헤더. 서버는 토큰의 SHA-256으로 `device`를 찾고, 그 태블릿에 연결된 어르신 한 명의 대화만 다룹니다.
- 태블릿 한 대당 연결 하나. 같은 기기에서 새로 연결하면 서버는 예전 연결을 닫습니다.
- 연결 직후 서버가 `connection.ready`를 보냅니다. 진행 중 세션이 있으면 `activeSession`에 들어 있어 재연결 후에도 대화를 이어갑니다.
- 30초 동안 아무 메시지가 없으면 어느 쪽이든 `ping`을 보내고, 받은 쪽은 같은 `nonce`로 `pong`을 돌려줍니다.

## 2. 프레임 규칙

**텍스트 프레임**은 모두 같은 봉투를 씁니다.

```json
{ "type": "ai.reply", "id": "m-12", "ts": "2026-10-06T07:18:02Z", "data": { ... } }
```

| 필드 | 필수 | 설명 |
| --- | --- | --- |
| `type` | 예 | 이벤트 이름 (아래 표) |
| `data` | 예 | 이벤트별 내용. 정의되지 않은 필드는 넣지 않습니다. |
| `id` | 아니오 | 보낸 쪽이 붙이는 메시지 ID. 서버가 그 메시지를 처리하지 못하면 `error.data.replyTo`로 돌려줍니다. |
| `ts` | 아니오 | 보낸 시각 (RFC 3339) |

**바이너리 프레임**은 오디오만 담습니다. `audio` 필드가 있는 텍스트 프레임 바로 다음에, 같은 쪽이 `audio.bytes` 길이의 바이너리 프레임을 **정확히 1개** 보냅니다.
서버는 쓰기를 한 고루틴에서만 하므로 텍스트와 바이너리 짝이 섞이지 않습니다. 태블릿도 같은 규칙을 지킵니다.

| 방향 | 오디오 형식 |
| --- | --- |
| 태블릿 → 서버 (`elder.audio`) | `pcm16` 16kHz 모노 기본. Clova STT가 받는 `opus`, `aac`도 허용 |
| 서버 → 태블릿 (`ai.reply`) | `mp3` (Clova TTS 출력 그대로) |

## 3. 이벤트

### 서버 → 태블릿

| 이벤트 | 언제 | 주요 필드 |
| --- | --- | --- |
| `connection.ready` | 연결 직후 한 번 | `deviceId`, `elderId`, `activeSession`, `openerVersion` |
| `session.started` | 세션 시작 (ETA 트리거, 말동무 버튼, 안부 일정) | `sessionId`, `mode`, `startedBy`, `caregiverEtaMinutes` |
| `session.rejected` | `session.request`를 받아들이지 않음 | `reason`: `ALREADY_ACTIVE`, `COMPANION_DISABLED`, `BEDTIME`, `TOKEN_LIMIT` |
| `session.ended` | 세션 종료 | `sessionId`, `reason` (DB `ended_reason`과 같은 값) |
| `elder.transcript` | 어르신 발화의 STT 결과 | `seq`, `utteranceId`, `clientId`, `text` |
| `ai.opener` | 0번 문장 선택 | `turnId`, `clipId`, `category` |
| `ai.filler` | 1번 문장이 늦을 때 한 번 | `turnId`, `clipId` |
| `ai.reply` | 스트리밍 문장 하나 | `turnId`, `index`(1부터), `utteranceId`, `text`, `audio` |
| `ai.turn_end` | 이 턴에 더 올 문장 없음 | `turnId`, `outcome`, `sentences` |
| `caregiver.eta` | 픽업 대기 중 남은 분이 바뀜 | `visitId`, `minutes` |
| `error` | 받은 메시지를 처리하지 못함 (연결은 유지) | `code`, `message`, `replyTo` |

### 태블릿 → 서버

| 이벤트 | 언제 | 주요 필드 |
| --- | --- | --- |
| `session.request` | 말동무 버튼을 누름 | `mode`: `COMPANION`만 |
| `session.end` | 어르신이 그만하기 버튼을 누름 | `sessionId` |
| `elder.audio` | 발화 한 구간이 끝남 (다음 프레임이 오디오) | `sessionId`, `clientId`, `audio` |
| `elder.text` | 텍스트 모드 발화 (단계 3 개발용 CLI) | `sessionId`, `clientId`, `text` |
| `elder.barge_in` | 재생 중 어르신이 말을 시작함 | `sessionId`, `turnId`, `playedIndex` |

### 양방향

| 이벤트 | 주요 필드 |
| --- | --- |
| `ping` | `nonce` |
| `pong` | `nonce` |

## 4. 번호 규칙

- `seq`/`turnId`는 세션 안의 턴 번호로 `utterance.seq`와 같습니다. 어르신 발화와 AI 응답이 번갈아 하나씩 번호를 받습니다(인사 0, 어르신 1, AI 2 …).
- `index`는 AI 턴 안의 문장 번호로 `utterance.chunk_index`와 같습니다. **0번은 언제나 `ai.opener`**(캐시 음성, Claude 호출 없음)이고 `ai.reply`는 1번부터입니다.
- 세션 첫 인사는 턴 0이며 0번 문장 없이 `ai.reply` 1번부터 옵니다.

## 5. 한 턴의 흐름

정상 턴 ([architecture.md](../../docs/architecture.md) 6.1절):

```
태블릿                                   서버
  │ elder.audio {clientId:u-3} + 바이너리 ─▶ │ STT → utterance(ELDER, seq 1) 저장
  │ ◀─ elder.transcript {seq:1}              │ 규칙 필터 통과
  │ ◀─ ai.opener {turnId:2, RECALL}          │ 0번 문장 선택 (규칙), 동시에 Claude 스트리밍 시작
  │   (캐시 음성 즉시 재생)                    │
  │ ◀─ ai.filler {turnId:2}                  │ (1번 문장이 늦을 때만)
  │ ◀─ ai.reply {turnId:2, index:1} + mp3    │ 문장 끝에서 잘라 TTS
  │ ◀─ ai.reply {turnId:2, index:2} + mp3    │
  │ ◀─ ai.turn_end {COMPLETED, sentences:2}  │
```

- **위급 감지**: 규칙 필터에 걸리면 `ai.opener {category: ESCALATION}`(안심 문장 음성) 다음 바로 `ai.turn_end {outcome: ESCALATED, sentences: 0}`를 보냅니다. Claude를 부르지 않습니다.
- **대체 문장**: Claude가 시간 안에 응답하지 않거나 오류가 나면 대체 문장을 `ai.reply`로 보내고 `ai.turn_end {outcome: FALLBACK}`로 끝냅니다.
- **끼어들기**: 태블릿은 어르신이 말을 시작하면 재생 큐를 비우고 `elder.barge_in`을 보냅니다. 서버는 그 턴의 남은 생성을 멈추고 `ai.turn_end {outcome: CANCELLED}`를 보냅니다. 끼어들기 뒤에 도착한 같은 턴의 `ai.reply`는 태블릿이 버립니다.
- **텍스트 모드**: `elder.text`는 STT만 건너뛰고 같은 흐름을 탑니다. `ai.reply.audio`는 `null`일 수 있습니다.

## 6. 세션 흐름

- **픽업 대기**: 조무사 위치(`POST /visits/{id}/location`)로 ETA가 기준 이내가 되면 서버가 방문을 `SESSION_ACTIVE`로 한 번만 차지하고 세션을 만든 뒤 `session.started {mode: PICKUP_BRIDGE, startedBy: SYSTEM}`을 보냅니다. 조무사가 도착하면(`POST /visits/{id}/arrive`) `session.ended {reason: CAREGIVER_ARRIVED}`.
- **말동무**: 태블릿이 `session.request {mode: COMPANION}`을 보내면 서버가 세션을 만들고 `session.started {mode: COMPANION, startedBy: ELDER}`, 안 되면 `session.rejected`. 보호자가 정한 안부 시각에는 서버가 먼저 `session.started {startedBy: SCHEDULE}`을 보냅니다. 무응답·취침 시간·토큰 한도·`session.end`로 끝납니다.
- 어르신 한 명에게 진행 중 세션은 하나뿐입니다(DB 부분 유니크 인덱스). 픽업 대기 중 말동무 버튼은 `session.rejected {reason: ALREADY_ACTIVE}`입니다.

## 7. 오류

| `code` | 뜻 |
| --- | --- |
| `INVALID_MESSAGE` | JSON이 아니거나 스키마에 맞지 않음 |
| `NO_ACTIVE_SESSION` | 세션이 없거나 이미 끝난 세션에 발화를 보냄 |
| `AUDIO_FRAME_MISSING` | `audio` 텍스트 프레임 다음에 바이너리 프레임이 오지 않았거나 길이가 다름 |
| `STT_FAILED` | 음성 인식 실패 |
| `INTERNAL` | 그 밖의 서버 오류 |
