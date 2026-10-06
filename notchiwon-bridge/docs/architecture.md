# 노치원 픽업 브릿지 아키텍처

> 기준일: 2026-10-05. 원본 제안서와 논의: [기술 스택 제안 문서](https://claude.ai/code/artifact/45f4d632-1be7-48ee-ac89-524ec03389a9)
> 실행 순서와 작업 방식은 [development-process.md](./development-process.md)를 참고하세요.

앱은 **Flutter**(어르신 태블릿 앱, 조무사 앱), 서버는 **Go**, DB는 **PostgreSQL + Redis**이고, Claude는 **서버에서만** 호출합니다.
기존 설계 원칙(관심사 키워드 시간 감쇠, 규칙 기반 위급 필터 우선, 대화는 Sonnet·요약은 Haiku)은 그대로 유지합니다.

## 1. 전체 구성

```
 ┌────────────────────┐                     ┌────────────────────┐
 │ 어르신 태블릿 앱     │                     │ 조무사 앱            │
 │ Flutter, 키오스크    │                     │ Flutter, 지도·위치   │
 └─────────┬──────────┘                     └─────────┬──────────┘
   WebSocket│ 음성 ↑ / 0번 문장·AI 응답·ETA ↓     REST 위치 ↑│ FCM 푸시·브리핑 ↓
 ┌─────────▼────────────────────────────────────────────▼──────────┐
 │ Go 서버 (단일 바이너리)                                              │
 │  대화 오케스트레이터 │ 안전 필터 │ 비동기 작업(River) │ 방문·ETA        │
 └──────┬──────────┬──────────┬──────────┬──────────┬──────────┬──────┘
   PostgreSQL    Redis     Claude API   Clova     카카오·티맵    FCM
   모든 기록   최근 대화    Sonnet·Haiku  STT·TTS    ETA 계산    푸시 알림
```

두 앱은 Go 서버하고만 통신하고, 어르신 발화는 항상 안전 필터를 거친 뒤에 Claude로 갑니다.

## 2. 레포 구조

```
notchiwon-bridge/
├── server/                 Go 서버
├── app/                    Flutter 앱 (태블릿, 조무사)
├── docs/                   아키텍처·실행 과정 문서
├── docker-compose.yml      postgres, redis, server
└── backend/                기존 NestJS (Go 서버가 기능을 따라잡으면 삭제)
```

## 3. Flutter 앱

```
app/
├── apps/elder_tablet/      어르신 태블릿 (키오스크)
├── apps/caregiver/         조무사 휴대폰
├── packages/api_client/    OpenAPI에서 생성한 Dart 클라이언트, WebSocket 클라이언트
├── packages/voice/         녹음·발화 구간 감지·재생 큐, 0번 문장 음성 캐시
└── packages/ui/            큰 글씨·고대비 테마
```

| 항목 | 어르신 태블릿 (`elder_tablet`) | 조무사 앱 (`caregiver`) |
| --- | --- | --- |
| 기기 | 어르신 댁에 둔 태블릿 1대 | 조무사 개인 휴대폰 |
| 핵심 기능 | 픽업 대기 대화, 하원 후 말동무, 도착 임박 안내 | 방문 목록, 지도·ETA, 브리핑 카드, 위급 알림 |
| 서버 통신 | `web_socket_channel` 상시 연결 | 생성된 REST 클라이언트(`http`), 위치는 10초 간격 |
| 음성 | `record`(녹음) + 발화 구간 감지, `just_audio`(재생 큐) | 없음 |
| 기기 제어 | Lock Task 모드·화면 상시 켜짐 (플랫폼 채널로 Android 네이티브 호출) | 일반 앱 |
| 백그라운드 | 마이크용 Foreground Service | `geolocator` + 위치용 Foreground Service |
| 푸시·지도 | 없음 | `firebase_messaging`, 카카오맵 또는 티맵 플러그인 |

상태 관리는 Riverpod, 화면 이동은 go_router를 기본으로 합니다.

## 4. Go 서버

모듈형 단일 바이너리입니다. 기존 NestJS 모듈 5개를 Go 패키지로 옮기고, 외부 서비스는 인터페이스 뒤에 둡니다.

```
server/
├── cmd/api/main.go          HTTP·WebSocket 서버 + River 워커
├── api/openapi.yaml         REST 계약 (앱과 공유)
├── db/
│   ├── migrations/          goose SQL 마이그레이션
│   └── queries/             sqlc 쿼리
└── internal/
    ├── visit/               방문, 위치 수신, ETA, 세션 트리거
    ├── session/             대화 오케스트레이터, WebSocket 핸들러
    ├── escalation/          규칙 필터, 위급 이벤트
    ├── opener/              0번 문장 분류·선택
    ├── briefing/            브리핑·하루 요약 작업
    ├── keyword/             관심사 태그 감쇠 갱신 작업
    ├── companion/           말동무 모드 일정·토큰 한도
    ├── llm/                 Claude 호출, 프롬프트 템플릿
    ├── speech/              STT·TTS (Clova)
    ├── eta/                 카카오모빌리티·티맵
    ├── notify/              FCM
    ├── auth/                기기 토큰, 조무사 로그인
    └── db/                  sqlc 생성 코드
```

| 용도 | 라이브러리 |
| --- | --- |
| 라우터 | `go-chi/chi/v5` |
| REST 코드 생성 | `oapi-codegen` (Dart 클라이언트는 `openapi-generator`의 `dart` 생성기) |
| DB | `jackc/pgx/v5` + `sqlc` |
| 마이그레이션 | `pressly/goose/v3` |
| WebSocket | `coder/websocket` |
| Redis | `redis/go-redis/v9` |
| 작업 큐 | `riverqueue/river` |
| Claude | `anthropics/anthropic-sdk-go` |
| 푸시 | `firebase.google.com/go/v4` |
| 설정 | `caarlos0/env` |
| 로깅 | 표준 `log/slog` |
| 테스트 | `testcontainers-go` |

Redis에는 대화 중 짧은 히스토리와 연결 상태만 두고, 기록은 모두 Postgres에 남깁니다.

## 5. 데이터베이스

기존 Prisma 스키마의 테이블 10개를 **테이블·컬럼 이름 그대로** goose 마이그레이션 `00001_init.sql`로 옮기고, 다음을 바꿉니다.

| 변경 | 이유 |
| --- | --- |
| 시각은 `timestamptz`, PK는 `uuid DEFAULT gen_random_uuid()` | 한국 시각과 UTC 혼동 방지 |
| 상태값은 Postgres ENUM 대신 `text` + `CHECK` | 값 추가 마이그레이션이 간단 |
| 방문 상태 전이는 조건부 `UPDATE ... WHERE status IN (...) RETURNING` 한 문장 | 위치가 동시에 두 번 와도 세션은 하나만 시작 |
| `utterance`에 `seq`, `chunk_index`, `opener_clip_id`, `model`, `latency_ms` | 대화 순서, 0번 문장 추적, 지연 측정 |
| `escalation_event`에 `source`(RULE / LLM), `rule_id` | 누가 감지했는지 기록해 규칙 튜닝 |
| `conversation_session.visit_id` nullable, `mode`(PICKUP_BRIDGE / COMPANION), `started_by`(SYSTEM / ELDER / SCHEDULE) | 하원 후 말동무 세션 |
| 새 테이블 `device` | 태블릿 인증, FCM 토큰 |
| 새 테이블 `caregiver_location` | ETA 트리거 검증, 짧게 보관 |
| 새 테이블 `opener_clip` | 0번 문장(유형, 문장, 목소리, 오디오 위치) |
| 새 테이블 `companion_schedule`, `daily_digest` | 말동무 안부 시각·취침 시간, 보호자용 하루 요약 |

음성 원본은 객체 저장소(NCP Object Storage 또는 S3 서울)에 두고 DB에는 `audio_ref`만 저장합니다.
키워드 감쇠 갱신은 `INSERT ... ON CONFLICT (elder_id, keyword) DO UPDATE SET score = keyword_tag.score * 0.9 + 1` 한 문장으로 처리합니다.

## 6. 에이전트 (대화 엔진)

Claude는 서버에서만 호출합니다. API 키가 기기에 남지 않고, 안전 필터를 건너뛸 길이 없으며, 모델·프롬프트를 앱 업데이트 없이 바꿀 수 있습니다.

### 6.1 한 턴의 흐름

1. 어르신 발화가 끝나면 태블릿이 음성을 WebSocket으로 보냅니다.
2. 서버가 Clova로 STT를 돌리고 `utterance`(ELDER)에 저장합니다.
3. **규칙 필터가 항상 먼저** 검사합니다. 걸리면 `escalation_event`를 만들고 FCM으로 알리며, Claude를 부르지 않고 정해진 안심 문장만 보냅니다.
4. 걸리지 않으면 LLM 없이 규칙으로 발화 유형(회상, 질문, 감정, 짧은 대답, 인사)을 분류해 **0번 문장**을 고르고 `ai.opener { clip_id }`를 보냅니다. 태블릿은 캐시된 음성을 바로 재생합니다.
5. 같은 순간 Claude(Sonnet)를 스트리밍으로 호출합니다. 프롬프트에 "방금 '<0번 문장>'이라고 말했으니 그다음 문장부터 1~3문장으로 이어서 답하라"를 넣습니다.
6. 스트리밍 텍스트를 문장 끝(`다.` `요.` `?` `!`)에서 자르고, 문장마다 TTS를 만들어 `ai.reply { index: 1, 2, 3... }`로 보냅니다.
7. 태블릿 재생 큐는 0번이 끝나면 1번, 2번 순으로 틉니다. 어르신이 말을 시작하면 남은 문장을 취소합니다.
8. 1번 문장이 늦으면 짧은 추임새를 한 번 더 틀고, 정해진 시간을 넘기거나 오류가 나면 대체 문장으로 마무리합니다.
9. Claude에는 `flag_concern(type, reason)` 도구를 줍니다. 규칙이 놓친 신호를 **2차로** 보고하는 용도이며 규칙을 대신하지 않습니다(`source=LLM`).

### 6.2 0번 문장

| 발화 유형 | 예시 |
| --- | --- |
| 회상·이야기 | "아이고, 그러셨어요." / "아, 그랬구나요." |
| 질문 | "음, 그건 말이죠." / "아, 그거요." |
| 감정 표현 | "그러셨구나, 마음이 그러셨어요." |
| 짧은 대답 | "네, 네." / "그럼요." |
| 인사 | "네, 안녕하세요." |
| 위급 감지 | "지금 많이 힘드시죠, 조금만 기다려주세요." (Claude 호출 없음) |

기쁨과 슬픔 모두에 어울리는 중립적인 맞장구로만 구성하고, 유형마다 여러 개를 돌려 씁니다. 음성은 어르신별 목소리(`preferred_tts_voice`)로 미리 합성해 태블릿에 내려받아 둡니다.
토큰 절감은 크지 않고, 핵심 이득은 어르신이 느끼는 대기 시간입니다.

### 6.3 프롬프트와 세션 상태

- 프롬프트는 `internal/llm/prompts/` 템플릿 파일로 관리하고 버전을 세션에 기록합니다.
- 고정 부분(역할·안전 규칙)은 프롬프트 캐싱을 쓰고, 가변 부분은 어르신 프로필과 `keyword_tag` 로테이션 결과입니다.
- 히스토리는 Redis에 최근 약 10턴, 그 이전은 Haiku 요약으로 넣습니다. 서버 재시작 시 `utterance`에서 복구합니다.
- 모델 ID는 환경변수로 둡니다. 대화는 Sonnet, 브리핑·키워드 추출·요약은 Haiku입니다.
- 브리핑과 키워드 추출은 세션 종료 시 River 작업으로 등록하고, Haiku에 JSON 스키마를 줘 구조화된 결과를 받습니다. Kiwi 형태소 분석 서브서비스는 MVP에서 뺍니다.

### 6.4 하원 후 말동무 모드

| 항목 | 픽업 대기 (`PICKUP_BRIDGE`) | 말동무 (`COMPANION`) |
| --- | --- | --- |
| 시작 | 조무사 ETA가 기준 이내가 되면 서버가 시작 | 어르신이 큰 버튼을 누르거나, 보호자가 정한 시각에 안부 대화 |
| 종료 | 조무사 도착 | 무응답, "그만할게" 같은 말, 취침 시간 |
| 위급 알림 | 조무사·센터 | 보호자 (센터 운영 시간에는 센터도) |
| 대화 후 요약 | 조무사 브리핑 카드 | 보호자용 하루 요약 |

말동무 모드는 버튼으로 시작하고 대화 중에만 마이크를 엽니다(집 안 사생활 보호). 어르신별 하루 토큰 한도를 두고, 넘으면 자연스럽게 마무리합니다.

### 6.5 WebSocket 이벤트 (`/ws/elder`)

정확한 메시지 형태와 규칙은 [server/api/ws-events.md](../server/api/ws-events.md)(JSON 스키마 `ws-events.schema.json`)가 기준입니다.

| 방향 | 이벤트 | 내용 |
| --- | --- | --- |
| 서버 → 태블릿 | `connection.ready` | 기기·어르신 ID, 진행 중 세션(재연결용), 0번 문장 목록 버전 |
| 서버 → 태블릿 | `session.started` / `session.rejected` / `session.ended` | 세션 ID·모드 / 말동무 요청 거절 사유 / 종료 사유 |
| 태블릿 → 서버 | `session.request` / `session.end` | 말동무 버튼 / 그만하기 버튼 |
| 태블릿 → 서버 | `elder.audio` | 발화 한 구간 (다음 바이너리 프레임이 오디오) |
| 태블릿 → 서버 | `elder.text` | 텍스트 모드 발화 (단계 3 개발용) |
| 태블릿 → 서버 | `elder.barge_in` | 재생 중 어르신이 말을 시작함 |
| 서버 → 태블릿 | `elder.transcript` | STT 결과 |
| 서버 → 태블릿 | `ai.opener` / `ai.filler` | 0번 문장 clip_id / 1번이 늦을 때 추임새 |
| 서버 → 태블릿 | `ai.reply` | 문장 번호(1부터), 텍스트, TTS 오디오 |
| 서버 → 태블릿 | `ai.turn_end` | 턴 종료 (정상, 대체 문장, 위급, 끼어들기) |
| 서버 → 태블릿 | `caregiver.eta` | 남은 분 |
| 서버 → 태블릿 | `error` | 처리하지 못한 메시지 |
| 양방향 | `ping` / `pong` | 연결 확인 |

조무사 앱은 REST로 충분합니다: `GET /visits/today`, `POST /visits/{id}/location`, `POST /visits/{id}/arrive`, `GET /sessions/{id}/briefing`, `POST /escalations/{id}/ack` 등. 전체는 [server/api/openapi.yaml](../server/api/openapi.yaml)입니다.

## 7. 확정된 결정 (2026-10-05)

| # | 결정 | 선택 |
| --- | --- | --- |
| 1 | 음성 처리 위치 | 서버 경유 (태블릿은 오디오만 주고받음) |
| 2 | 앱 프레임워크 | Flutter, 한 저장소에 앱 2개 + 공유 패키지 |
| 3 | DB 접근 | sqlc + pgx |
| 4 | 마이그레이션 | goose |
| 5 | API 계약 | OpenAPI 먼저, Go·Dart 코드 생성 |
| 6 | 비동기 작업 | River (Postgres) |
| 7 | 기존 NestJS | PR #1은 닫고 동작만 Go 명세로, `backend/`는 Go가 따라잡으면 삭제 |
| 8 | 말동무 시작 방식 | 버튼 시작, 대화 중에만 듣기 + 보호자 지정 시각 안부 |
| 9 | 0번 문장 선택 | 서버가 STT 후 규칙으로 유형 분류, 태블릿은 캐시 음성 재생 |
| 10 | Dart REST 클라이언트 (2026-10-06) | `openapi-generator`의 `dart`(http) 생성기. `dart-dio`는 build_runner가 필요해 생성 단계가 늘어남 |
| 11 | Go 버전 (2026-10-06) | 1.25 (pgx v5.11이 요구) |
| 12 | 진행 중 세션 (2026-10-06) | 어르신당 하나 (`conversation_session` 부분 유니크 인덱스). 픽업 대기 중 말동무 요청은 거절 |
| 13 | 말동무 중 픽업 시작 (2026-10-06) | 픽업 대기가 우선. 진행 중 말동무 세션은 `PREEMPTED`로 끝내고 픽업 대기 세션 시작 |
| 14 | 조무사 임시 로그인 (2026-10-06) | 센터 발급 아이디·비밀번호(bcrypt) + HMAC 서명 액세스 토큰(`AUTH_SECRET`). 로그인 방식이 정해지면 교체 |
| 15 | DB 통합 테스트 (2026-10-06) | `testcontainers-go` 대신 `TEST_DATABASE_URL` 서버에 테스트마다 임시 DB. CI는 Postgres 서비스 컨테이너 사용 |

## 8. 아직 정하지 않은 것

- 대화 원문·음성 보관 기간과 암호화 (개인정보 검토 필요)
- 배포 위치: NCP vs AWS 서울 리전
- 조무사 로그인 방식: 전화번호 인증 vs 센터 발급 계정
- 위급 키워드 목록의 요양·의료 전문가 검토
