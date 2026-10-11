# 명세: 계약 (REST · WebSocket · DB · 코드 생성)

## 목적

서버와 앱 세 개가 서로 기대는 모양을 한곳에 고정합니다. 계약 파일이 기준이고, 코드는 거기서 생성하거나 거기에 맞춰 손으로 씁니다.

## 경계

- 한다: API 경로와 스키마, WebSocket 이벤트와 프레임 규칙, 테이블과 제약, 생성 코드 갱신 규칙.
- 하지 않는다: 동작 규칙(언제 세션이 시작되는지 등)은 각 서버 명세가 정합니다.

## 파일

| 계약 | 원본 | 생성되는 것 | 검사 |
| --- | --- | --- | --- |
| REST | `server/api/openapi.yaml` | `server/internal/apigen/apigen.gen.go` (oapi-codegen, strict 서버), `app/packages/api_client/lib/src/generated/` (openapi-generator dart) | CI `generate` 작업이 `make generate` 뒤 차이가 없는지 확인 |
| WebSocket | `server/api/ws-events.md`(설명), `ws-events.schema.json`(형태 기준), `ws-events.examples.json` | 없음. 서버는 스키마로 들어오는 프레임을 검사, Dart는 `api_client/lib/src/ws/events.dart`를 손으로 관리 | `go test ./api` (예시가 스키마에 맞는지) |
| DB | `server/db/migrations/0000N_*.sql` (goose) | `server/internal/db/*.sql.go` (sqlc, 쿼리는 `server/db/queries/`) | `internal/db/schema_test.go` (실제 Postgres) |

## 인터페이스

### REST 경로 (인증별)

| 인증 | 경로 |
| --- | --- |
| 없음 | `GET /healthz`, `POST /auth/caregiver/login`, `POST /auth/guardian/login` |
| 조무사 토큰 (`CaregiverAuth`) | `GET /visits/today`, `GET /visits/{id}`, `POST /visits/{id}/location`, `POST /visits/{id}/arrive`, `GET /sessions/{id}/briefing`, `POST /sessions/{id}/briefing/read`, `GET /escalations/open`, `GET /escalations/{id}`, `POST /escalations/{id}/ack`, `PUT /devices/me/fcm-token`, `GET·PUT /elders/{id}/companion-schedule`, `GET /elders/{id}/daily-digests/{date}` |
| 보호자 토큰 (`GuardianAuth`) | `GET /guardian/me`, `PUT /guardian/devices/me/fcm-token`, `GET·PUT /guardian/elders/{id}/companion-schedule`, `GET /guardian/elders/{id}/daily-digests`, `GET /guardian/escalations/open`, `GET /guardian/escalations/{id}`, `POST /guardian/escalations/{id}/ack` |
| 기기 토큰 (`DeviceAuth`) | `GET /tablet/me`, `GET /tablet/opener-clips`, `GET /tablet/opener-clips/{clipId}/audio`, `GET /ws/elder` (WebSocket, 같은 토큰) |

오류 본문은 항상 `{"code": "...", "message": "..."}`(`ApiError`)이고, 메시지는 사용자에게 보여 줄 수 있는 한국어입니다.

### WebSocket 이벤트

| 방향 | 이벤트 |
| --- | --- |
| 서버 → 태블릿 | `connection.ready`, `session.started`, `session.rejected`, `session.ended`, `elder.transcript`, `ai.opener`, `ai.filler`, `ai.reply`(+MP3 바이너리), `ai.turn_end`, `caregiver.eta`, `error` |
| 태블릿 → 서버 | `session.request`, `session.end`, `elder.text`, `elder.audio`(+PCM 바이너리), `elder.barge_in`, `client.metrics` |
| 양방향 | `ping`, `pong` |

## 요구사항

- `CTR-01`: 계약을 바꾸면 생성 코드도 같은 PR에서 갱신합니다 (`cd server && make generate`, Java와 Dart 필요).
- `CTR-02`: 머지된 마이그레이션 파일은 고치지 않고 새 번호 파일을 추가합니다.
- `CTR-03`: 텍스트 프레임은 봉투 `{type, id?, ts?, data}`입니다. `data`에 정의되지 않은 필드는 넣지 않습니다 (스키마가 거절).
- `CTR-04`: `audio`가 있는 텍스트 프레임 바로 다음에 그 길이의 바이너리 프레임이 정확히 1개 옵니다. 서버는 쓰기를 한 고루틴에서만 해서 짝이 섞이지 않습니다.
- `CTR-05`: 서버는 들어오는 모든 텍스트 프레임을 `ws-events.schema.json`으로 검사하고, 맞지 않으면 `error {code: INVALID_MESSAGE, replyTo: id}`를 보냅니다.
- `CTR-06`: 턴 번호(`turnId`)는 AI 턴의 `utterance.seq`이고, 문장 번호(`index`)는 `chunk_index`입니다. 0번은 `ai.opener`, 1번부터 `ai.reply`, 추임새는 번호가 없습니다.
- `CTR-07`: 어르신 한 명에게 열린 세션은 하나입니다 (`conversation_session_one_open_per_elder_idx`). 세션은 끝나면 `ended_at`과 `ended_reason`이 함께 채워집니다 (CHECK 제약).
- `CTR-08`: 비밀값(토큰, 비밀번호)은 해시만 저장합니다 (기기 토큰 SHA-256, 비밀번호 bcrypt).

## 설정값

없음 (계약 파일 자체).

## 실패할 때

생성 코드가 커밋과 다르면 CI `generate`가 빨갛게 됩니다. 앱이 모르는 WebSocket `type`을 받으면 `UnknownEvent`로 무시하고, 서버가 모르는 값을 enum으로 보내면 Dart 쪽은 기본값(`RECALL`, `COMPANION`)으로 읽습니다.

## 확장 지점

- 새 REST 경로: `openapi.yaml`에 추가 → `make generate` → `httpapi.Server`에 생성된 인터페이스 메서드 구현 (컴파일러가 빠진 메서드를 알려 줌).
- 새 WebSocket 이벤트: `ws-events.md` 표, `ws-events.schema.json`, `ws-events.examples.json`, 서버 `ws.go`의 `clientEvents`·`handle`, Dart `events.dart`의 `decode`나 `ElderCommand`를 함께 고칩니다.
- 새 테이블: 새 마이그레이션 + `db/queries/*.sql` + `make generate`.

## 알려진 문제

- [CR-06](../code-review.md#cr-06): `elder.barge_in.playedIndex`를 서버가 쓰지 않음.

## 다시 만들 때

- 앱이 이미 배포된 뒤에는 필드 삭제·이름 변경이 깨지는 변경입니다. 새 필드 추가와 새 이벤트 추가만 하위 호환입니다.
- WebSocket 인증은 HTTP 헤더(`Authorization: Bearer`)로 합니다. 브라우저 클라이언트를 붙이려면 쿼리 토큰이나 첫 메시지 인증을 따로 설계해야 합니다.
