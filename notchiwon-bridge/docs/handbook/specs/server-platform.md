# 명세: 서버 기반 (cmd · config · auth · httpapi · jobs · migrate · db)

## 목적

다른 모듈이 올라앉는 바탕입니다. 설정을 읽고, DB와 작업 큐를 띄우고, 인증한 요청을 각 서비스로 넘기고, 운영자 명령을 제공합니다. 동작 규칙은 없고 연결만 합니다.

## 경계

- 한다: 프로세스 시작·종료, 환경변수 검사, 마이그레이션, 토큰 발급·검증, REST 라우팅과 오류 모양, River 큐 구성, 운영자 CLI.
- 하지 않는다: 대화·픽업·말동무·위급의 규칙(각 명세), 계약 정의(→ [contracts.md](./contracts.md)).

## 파일

| 파일 | 내용 |
| --- | --- |
| `server/cmd/api/main.go` | 의존성 조립: DB 풀 → 마이그레이션 → LLM·음성·지도·FCM(키 없으면 `Unavailable`) → 서비스 → `session.Hub` → River 클라이언트 → HTTP 서버. `Set*`로 순환 의존을 끊음 |
| `server/cmd/admin/main.go` | 운영자 명령: `create-tablet`, `set-caregiver-login`, `set-guardian-login`, `seed-demo`, `synth-openers` |
| `server/cmd/elder-cli/` | 태블릿 없이 `/ws/elder`에 붙는 글자 클라이언트 (개발·시연용) |
| `server/internal/config/config.go` | `Config` (env 태그), `Load`의 값 검사 |
| `server/internal/auth/auth.go` | `Tokens`(HMAC-SHA256, 접두어로 조무사·보호자 구분), bcrypt, `CheckPasswordOrDummy`, 컨텍스트 주인(`WithCaregiver` 등), `BearerToken` |
| `server/internal/auth/tablet.go` | 기기 토큰 만들기, SHA-256 해시, `LookupTablet` |
| `server/internal/httpapi/router.go` | chi 라우터, `/ws/elder` 장착, strict 핸들러, `authenticate` 미들웨어, `handleError`·`writeError` |
| `server/internal/httpapi/server.go`, `guardian.go` | 생성된 `apigen.StrictServerInterface` 구현 (조무사·태블릿 / 보호자) |
| `server/internal/jobs/jobs.go` | 작업 종류(`*Args`), 큐, `NewClient`, `NewInsertOnly`, `Enqueuer` |
| `server/internal/migrate/migrate.go` | goose(내장 SQL) + River 자체 마이그레이션 |
| `server/db/migrations/00001~00007` | 스키마 (→ [whitebox.md](../whitebox.md#7-데이터-모델) ER) |
| `server/internal/db/` | sqlc 생성 코드, `schema_test.go` |
| `server/internal/testdb/` | 테스트용 Postgres 준비 (`TEST_DATABASE_URL`) |
| 테스트 | `auth/auth_test.go`, `httpapi/router_test.go`, `stage5_test.go`, `stage6_test.go`, `config` 검사는 `Load` 호출부 |

## 인터페이스

| 방향 | 이름 | 설명 |
| --- | --- | --- |
| 제공 | `config.Load() (Config, error)` | 필수값 없음·잘못된 시간 설정·없는 시간대면 시작하지 않음 |
| 제공 | `auth.Tokens.IssueCaregiver/IssueGuardian`, `VerifyCaregiver/VerifyGuardian` | 토큰 `"<접두어>.<계정 ID>.<만료 unix>.<서명>"` |
| 제공 | `auth.LookupTablet(ctx, q, token)` | 기기 토큰 해시로 태블릿·어르신 찾기 |
| 제공 | `jobs.Enqueuer` | `session.Alerter`, `visit.SessionNotifier`(브리핑), `session.Compactor`, `companion.Queue`를 구현 |
| 제공 | `httpapi.NewRouter` | `http.Handler` 하나 |
| 기대 | 각 서비스 (`visit.Service`, `companion.Policy`, `session.Hub` 등) | `httpapi.Deps`로 주입 |

## 요구사항

- `PLAT-01`: 서버는 `DATABASE_URL`과 32바이트 이상의 `AUTH_SECRET` 없이 시작하지 않습니다. `CONVERSATION_TURN_TIMEOUT < CONVERSATION_FIRST_SENTENCE_TIMEOUT`이나 없는 `TIME_ZONE`도 시작 오류입니다.
- `PLAT-02`: `MIGRATE_ON_START=true`(기본)면 시작할 때 goose와 River 마이그레이션을 모두 올립니다. 실패하면 시작하지 않습니다.
- `PLAT-03`: 외부 키(Anthropic, AWS 음성, 카카오, FCM)가 없어도 서버는 뜹니다. 각자 `Unavailable` 구현이 들어가고 기능은 대체 동작을 합니다 (각 명세의 "실패할 때").
- `PLAT-04`: 인증은 `openapi.yaml`의 보안 스킴이 정합니다. 미들웨어는 스킴에 맞는 토큰만 받고, 다른 종류 토큰(예: 보호자 토큰으로 조무사 경로)은 401입니다.
- `PLAT-05`: 401 본문은 `UNAUTHENTICATED`, 처리 중 오류는 `INTERNAL` + 고정 한국어 메시지이며 원인은 로그(`request_id` 포함)에만 남깁니다. 인증 중 DB 오류는 401이 아니라 500입니다.
- `PLAT-06`: 로그인은 계정이 없어도 bcrypt 비교를 한 번 해서(`CheckPasswordOrDummy`) 응답 시간으로 계정 존재를 알 수 없게 합니다.
- `PLAT-07`: 조무사·보호자 토큰은 `CAREGIVER_TOKEN_TTL`(기본 12시간) 뒤 만료됩니다. 갱신 토큰은 없고 다시 로그인합니다.
- `PLAT-08`: 기기 토큰은 `cmd/admin create-tablet`이 한 번만 보여 주고 DB에는 SHA-256만 남깁니다 (CTR-08).
- `PLAT-09`: River 큐는 `alerts`(4), `briefings`(2), `companion`(2)로 나눠, 느린 브리핑이 위급 알림을 막지 않게 합니다. 주기 작업은 `companion_tick`(1분, 시작 시 한 번) 하나입니다.
- `PLAT-10`: 서버를 여러 대 띄워도 됩니다. 주기 작업은 River 리더만 돌리고, 세션 하나 규칙은 DB가 지킵니다 (PICK-02). 단, 태블릿 연결은 붙은 인스턴스에만 있습니다 (아래 알려진 문제).
- `PLAT-11`: `SIGINT`/`SIGTERM`이면 HTTP 서버와 River를 정해진 시간 안에 멈추고 나갑니다.

## 설정값

| 환경변수 | 기본 | 뜻 |
| --- | --- | --- |
| `PORT` | `8000` | HTTP 포트 |
| `LOG_LEVEL` | `info` | slog 수준 |
| `DATABASE_URL` | 필수 | Postgres |
| `REDIS_URL` | 없음 | 읽기만 하고 쓰지 않음 ([CR-13](../code-review.md#cr-13)) |
| `MIGRATE_ON_START` | `true` | 시작 시 마이그레이션 |
| `AUTH_SECRET` | 필수 | 토큰 서명 키 (32바이트 이상) |
| `CAREGIVER_TOKEN_TTL` | `12h` | 조무사·보호자 토큰 수명 |
| `TIME_ZONE` | `Asia/Seoul` | 서버 기본 시간대 |

그 밖의 값은 모듈 명세의 설정값 표에 있습니다 (대화: [server-conversation.md](./server-conversation.md), 픽업: [server-pickup.md](./server-pickup.md), 말동무: [server-companion.md](./server-companion.md), 안전: [server-safety.md](./server-safety.md)). 로컬 값 예시는 저장소 루트의 `.env.example`이고 `scripts/dev.sh`가 `.env`를 읽습니다.

## 실패할 때

| 상황 | 동작 |
| --- | --- |
| 설정 오류 | 시작하지 않고 어느 값이 틀렸는지 출력 |
| DB 연결 실패 | 시작하지 않음 |
| 작업 등록 실패 | 호출한 쪽이 오류 로그 (요청 자체는 성공, [CR-05](../code-review.md#cr-05)) |
| 요청 본문이 스키마와 다름 | 400 `INVALID_ARGUMENT` |

## 확장 지점

- 새 환경변수: `Config`에 필드 + `env` 태그 + 루트 `.env.example` + 이 표.
- 새 인증 종류: `openapi.yaml` 보안 스킴 → `make generate` → `authenticate`에 `case` + `auth`에 `With*`/`*From`.
- 새 작업: `jobs`에 `*Args`(`Kind`, `InsertOpts`의 큐·유일성) + 워커를 `main.go`의 `river.Workers`에 등록 + `Enqueuer` 메서드.
- 운영자 명령: `cmd/admin`의 `switch`에 한 줄.

## 알려진 문제

- [CR-10](../code-review.md#cr-10): 로그인에 횟수 제한이 없음.
- [CR-12](../code-review.md#cr-12): 말동무 설정을 쓰는 길이 둘(조무사·보호자), 오래된 주석.
- [CR-13](../code-review.md#cr-13): Redis 설정은 있지만 쓰지 않음.
- 태블릿 연결이 인스턴스 메모리(`Hub.conns`)에 있어서, 서버를 2대 이상 띄우면 다른 인스턴스에서 연 세션을 태블릿이 즉시 받지 못합니다 (재접속 때 `connection.ready.activeSession`으로는 받음). 파일럿은 1대 기준.

## 다시 만들 때

- `main.go`는 조립만 합니다. 규칙을 여기에 넣지 않습니다.
- 순환(엔진 ↔ 허브 ↔ 큐)은 생성자 인자가 아니라 `Set*`로 나중에 꽂습니다. 다시 만들 때 인터페이스를 소비하는 쪽 패키지에 두면 같은 구조가 나옵니다.
- 트랜잭션과 작업 등록을 묶으려면 `river.Client.InsertTx`를 쓰면 됩니다 (지금은 안 씀, CR-05).
