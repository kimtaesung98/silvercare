# 화이트박스: 시스템 속 구조

> 기준일: 2026-10-11. 그림은 GitHub에서 바로 그려지는 Mermaid입니다. 각 그림 아래 한 줄이 읽는 법입니다.
> 상자 이름은 실제 패키지·파일 이름이라 그대로 검색하면 코드가 나옵니다.

## 1. 전체 구성

```mermaid
flowchart LR
  subgraph 어르신댁["어르신 댁"]
    T["어르신 태블릿<br/>app/apps/elder_tablet"]
  end
  subgraph 휴대폰["휴대폰"]
    C["조무사 앱<br/>app/apps/caregiver"]
    G["보호자 앱<br/>app/apps/guardian"]
  end
  subgraph 서버["Go 서버 (server/cmd/api, 단일 바이너리)"]
    WS["/ws/elder<br/>session.Hub"]
    REST["REST<br/>httpapi"]
    JOBS["작업 큐<br/>jobs (River)"]
  end
  PG[("PostgreSQL<br/>기록 + River 큐")]
  CL["Claude API<br/>Sonnet 대화 · Haiku 요약"]
  CV["Naver Clova<br/>STT · TTS (선택)"]
  KK["카카오모빌리티<br/>ETA (선택)"]
  FCM["FCM 푸시 (선택)"]

  T <-- "WebSocket: 발화 ↑ / 0번 문장·답·ETA ↓" --> WS
  C -- "REST: 방문·위치·도착·브리핑" --> REST
  G -- "REST: 하루 소식·위급·설정" --> REST
  WS --> PG
  REST --> PG
  JOBS --> PG
  WS --> CL
  WS -.-> CV
  REST -.-> KK
  JOBS --> CL
  JOBS -.-> FCM
  FCM -.-> C
  FCM -.-> G
```

앱 세 개는 서버하고만 통신합니다. 점선은 키가 없으면 대체 동작으로 바뀌는 외부 서비스입니다 (키별 동작은 [status.md](./status.md) 4절). Redis는 docker-compose에 있지만 코드는 아직 쓰지 않습니다.

## 2. 서버 패키지 의존 관계

```mermaid
flowchart TB
  main["cmd/api/main.go<br/>조립"]
  httpapi["httpapi<br/>REST 핸들러·인증"]
  hub["session.Hub / conn<br/>WebSocket"]
  engine["session.Engine<br/>대화 턴"]
  visit["visit.Service<br/>방문·위치·도착"]
  companion["companion<br/>정책·분 단위 tick·하루 소식·요약"]
  briefing["briefing.Worker"]
  notify["notify<br/>위급 푸시 Worker·FCM"]
  jobs["jobs<br/>River 인자·Enqueuer"]
  escalation["escalation<br/>규칙 필터"]
  opener["opener<br/>0번 문장 분류·목록"]
  llm["llm<br/>Claude 클라이언트·프롬프트"]
  speech["speech<br/>Clova STT·TTS"]
  eta["eta<br/>Schedule · Kakao"]
  auth["auth<br/>토큰·비밀번호·기기"]
  db["db<br/>sqlc 생성 쿼리"]

  main --> httpapi & hub & engine & visit & companion & briefing & notify & jobs
  httpapi --> visit & auth & opener & speech & db
  hub --> engine & opener & speech & auth & db
  engine --> escalation & opener & llm & db
  visit --> eta & db
  companion --> jobs & llm & notify & db
  briefing --> llm & jobs & db
  notify --> jobs & db
  jobs --> db
```

화살표는 import 방향입니다. `session`은 `visit`·`companion`·`jobs`를 import하지 않고, 그쪽이 `session`이 정의한 작은 인터페이스(`Alerter`, `Budget`, `Compactor`, `EndListener`)를 구현합니다. `main.go`가 `SetAlerter`·`SetBudget`·`SetCompactor`·`AlsoNotify`로 이어 붙입니다.

## 3. 픽업 브릿지: 위치 전송부터 브리핑까지

```mermaid
sequenceDiagram
  autonumber
  participant C as 조무사 앱
  participant V as visit.Service
  participant H as session.Hub
  participant T as 태블릿
  participant E as session.Engine
  participant J as jobs / briefing

  loop 10초마다
    C->>V: POST /visits/{id}/location
    V->>V: ETA 계산 (eta.Kakao 또는 Schedule)
    alt ETA ≤ 15분, 세션 없음
      V->>V: 한 트랜잭션: 방문 선점 · 말동무 세션 PREEMPTED · 픽업 세션 생성
      V->>H: SessionStarted (커밋 후)
      H->>T: session.started
      H->>E: Greet (0번째 턴 인사)
    else 세션 진행 중
      V->>H: EtaUpdated
      H->>T: caregiver.eta (분이 바뀔 때만)
    end
  end
  C->>V: POST /visits/{id}/arrive
  V->>V: 방문 COMPLETED · 세션 CAREGIVER_ARRIVED
  V->>H: SessionEnded → T: session.ended
  V->>J: SessionEnded → BriefingArgs 작업
  J->>J: Haiku 요약 → briefing_report
  C->>J: GET /sessions/{id}/briefing
```

세션은 동시에 위치가 20건 와도 하나만 생깁니다 (`ClaimVisitSession` 조건부 UPDATE와 `conversation_session_one_open_per_elder_idx`). 알림은 모두 트랜잭션 커밋 뒤에 갑니다.

## 4. 대화 한 턴 (서버 안)

```mermaid
flowchart TB
  in["발화 도착<br/>elder.text 또는 elder.audio→STT"] --> cancel["진행 중인 턴 취소<br/>(sessionState.begin)"]
  cancel --> save["ELDER 발화 저장<br/>(취소돼도 저장)"]
  save --> rule{"escalation.Check<br/>규칙 필터"}
  rule -- 걸림 --> esc["escalation_event 저장<br/>→ 푸시 작업"] --> reassure["ESCALATION 0번 문장<br/>또는 안심 문장<br/>Claude 부르지 않음"] --> endT["ai.turn_end"]
  rule -- 통과 --> mode{"말동무 모드?"}
  mode -- "그만하자 / 토큰 소진" --> bye["작별 문장 → 세션 종료"] --> endT
  mode -- 계속 --> op["0번 문장 고르기<br/>opener.Classify → Pick"]
  op --> claude["Claude 스트리밍<br/>문장 단위로 잘라 1번, 2번…"]
  claude -- "2초 안에 1번 없음" --> filler["추임새 ai.filler"]
  claude -- "6초 안에 1번 없음 / 실패" --> fb["대체 문장"]
  claude --> concern["flag_concern → LLM 위급 이벤트"]
  claude --> endT
  fb --> endT
```

규칙 필터가 항상 먼저이고, 걸리면 Claude는 호출되지 않습니다. 시간 값은 `CONVERSATION_FILLER_AFTER`(2초), `CONVERSATION_FIRST_SENTENCE_TIMEOUT`(6초), `CONVERSATION_TURN_TIMEOUT`(20초)입니다.

## 5. 태블릿 속 음성 경로

```mermaid
flowchart LR
  subgraph 듣기["SpeechInput (packages/voice)"]
    D["DeviceSpeechInput<br/>기기 음성 인식<br/>SPEECH=device (기본)"]
    R["UtteranceRecorder<br/>녹음 + 발화 구간 감지<br/>SPEECH=server"]
  end
  SC["SessionController<br/>elder_tablet"]
  WS["ElderSocket<br/>api_client"]
  subgraph 말하기["재생"]
    Q["PlaybackQueue<br/>0번→1번→2번"]
    SP["SpeakingPlayer"]
    JA["JustAudioPlayer<br/>MP3·캐시 파일"]
    TTS["FlutterTtsSpeaker<br/>기기 TTS"]
  end
  D -- "HeardText" --> SC
  R -- "HeardAudio" --> SC
  SC -- "elder.text / elder.audio" --> WS
  WS -- "ai.opener / ai.reply" --> SC
  SC --> Q --> SP
  SP -- "음성 있음" --> JA
  SP -- "음성 없음" --> TTS
  SC -. "AI가 말하는 동안 muted" .-> D
```

`SPEECH`는 빌드할 때 정합니다 (`scripts/dev.sh tablet`은 Clova 키가 있으면 `server`). AI 목소리는 설정이 없고, 서버가 보낸 MP3나 캐시된 0번 문장이 있으면 그것을, 없으면 기기 TTS를 씁니다.

## 6. 상태 전이

### 방문 (`visit.status`)

```mermaid
stateDiagram-v2
  [*] --> SCHEDULED
  SCHEDULED --> EN_ROUTE: 첫 위치 전송
  EN_ROUTE --> SESSION_ACTIVE: ETA ≤ 기준 (같은 위치 전송 안에서)
  SCHEDULED --> COMPLETED: 도착
  EN_ROUTE --> COMPLETED: 도착
  SESSION_ACTIVE --> COMPLETED: 도착
  SCHEDULED --> CANCELLED: (API 없음, 관리자 DB 작업)
  COMPLETED --> [*]
  CANCELLED --> [*]
```

### 대화 세션 (`conversation_session`)

```mermaid
stateDiagram-v2
  [*] --> 열림: 픽업(ETA) / 말동무 버튼(ELDER) / 안부 일정(SCHEDULE)
  열림 --> 종료: CAREGIVER_ARRIVED (픽업)
  열림 --> 종료: ELDER_DECLINED (그만하기 버튼·"그만하자")
  열림 --> 종료: TOKEN_LIMIT
  열림 --> 종료: NO_RESPONSE (5분 무응답)
  열림 --> 종료: BEDTIME
  열림 --> 종료: PREEMPTED (말동무 중 픽업 시작)
  종료 --> [*]
```

어르신 한 명에게 열린 세션은 하나뿐입니다 (부분 유니크 인덱스). 종료되면 `ended_at`과 `ended_reason`이 채워지고 다시 열리지 않습니다.

## 7. 데이터 모델

```mermaid
erDiagram
  daycare_center ||--o{ elder : "다님"
  daycare_center ||--o{ caregiver : "소속"
  guardian ||--o{ elder : "보호"
  elder ||--o{ visit : ""
  caregiver ||--o{ visit : "담당"
  visit ||--o| conversation_session : "픽업 세션"
  elder ||--o{ conversation_session : ""
  conversation_session ||--o{ utterance : "seq·chunk_index"
  opener_clip ||--o{ utterance : "0번 문장"
  opener_clip ||--o| opener_clip_audio : "MP3"
  conversation_session ||--o| briefing_report : ""
  conversation_session ||--o{ escalation_event : ""
  utterance ||--o{ escalation_event : "근거 발화"
  elder ||--o{ keyword_tag : "관심사 (감쇠)"
  elder ||--o| companion_schedule : ""
  elder ||--o{ daily_digest : "날짜별 1개"
  elder ||--o{ device : "태블릿"
  caregiver ||--o{ device : "휴대폰"
  guardian ||--o{ device : "휴대폰"
  caregiver ||--o{ caregiver_location : ""
```

원본은 `server/db/migrations/00001_init.sql`부터 `00007`까지입니다. River 테이블은 서버 시작 때 goose 다음에 따로 만듭니다.

## 8. 비동기 작업

```mermaid
flowchart LR
  subgraph alerts["큐 alerts (4)"]
    EA["escalation_alert<br/>notify.EscalationWorker"]
  end
  subgraph briefings["큐 briefings (2)"]
    BR["briefing<br/>briefing.Worker"]
    HS["history_summary<br/>companion.HistoryWorker"]
  end
  subgraph companionQ["큐 companion (2)"]
    TK["companion_tick (1분, 리더만)<br/>companion.TickWorker"]
    DD["daily_digest<br/>companion.DigestWorker"]
  end
  engine["session.Engine"] -- "위급 이벤트" --> EA
  engine -- "긴 대화" --> HS
  visit["visit / Hub.End"] -- "픽업 세션 종료" --> BR
  TK -- "취침 시작 또는 21:00" --> DD
  TK -- "안부 시각" --> start["말동무 세션 시작"]
  TK -- "무응답·취침" --> stop["세션 종료"]
```

괄호 안 숫자는 큐별 동시 작업 수입니다 (`jobs.NewClient`). 위급 알림은 자기 큐를 써서 브리핑이 밀려도 늦지 않습니다.
