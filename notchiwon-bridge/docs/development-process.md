# 노치원 픽업 브릿지 개발 과정

> 기준일: 2026-10-05. 무엇을 만드는지는 [architecture.md](./architecture.md)를 참고하세요.

[architecture.md](./architecture.md)의 구조를 **8단계**로 나눠 만듭니다. 각 단계는 끝났는지 확인할 수 있는 완료 기준이 있고, 앞 단계의 완료 기준을 통과해야 다음 단계로 넘어갑니다.
계약(API·DB·이벤트)을 먼저 확정하고, 대화 엔진은 텍스트로 먼저 검증한 뒤 음성을 붙입니다.

## 1. 단계별 계획

| 단계 | 이름 | 핵심 산출물 | 완료 기준 |
| --- | --- | --- | --- |
| 0 | 기반 정리 | `server/`, `app/` 뼈대, CI, docker-compose 갱신 | 빈 Go·Flutter 프로젝트에서 CI가 통과 |
| 1 | 계약 확정 | `openapi.yaml`, `00001_init.sql`, WebSocket 이벤트 명세 | 코드 생성(sqlc, oapi-codegen, Dart 클라이언트)이 오류 없이 돌고 주인이 검토 |
| 2 | 서버 뼈대 | 설정, DB, 마이그레이션, 방문·위치 API, 세션 트리거, 기기 인증 | 실제 Postgres로 통합 테스트 통과, 동시 위치 전송에도 세션 1개 |
| 3 | 대화 엔진 (텍스트) | 규칙 필터, 0번 문장 선택, Claude 스트리밍, 문장 분할, 대체 문장, `flag_concern` | 텍스트 WebSocket 클라이언트로 대화가 이어지고 PR #1의 테스트 케이스가 Go로 통과 |
| 4 | 태블릿 앱 | Flutter 대화 화면, 재생 큐, 0번 문장 캐시, Clova STT·TTS, 키오스크 | 실제 태블릿에서 음성 대화 동작, 발화 종료 후 첫 소리까지 시간 측정 |
| 5 | 조무사 앱과 비동기 작업 | 방문 목록, 위치 전송, FCM 위급 알림, 브리핑 카드, 브리핑·키워드 River 작업 | 위치 이동으로 세션이 시작되고 도착 시 브리핑 카드가 뜨는 흐름이 처음부터 끝까지 동작 |
| 6 | 말동무 모드 | 세션 모드, 버튼 시작, 안부 일정, 보호자 알림, 하루 요약, 토큰 한도, 히스토리 요약 | 방문 없이 대화가 시작·종료되고 보호자에게 하루 요약이 감 |
| 7 | 파일럿 준비 | 위급 키워드 전문가 검토, 보관 정책, 배포, 모니터링 | 파일럿 센터 1곳에서 실제 기기로 운영 가능 |

단계 3이 끝나 Go 서버가 기존 NestJS 기능을 모두 갖추면 `backend/`를 삭제합니다.

## 2. 단계별 세부 작업

### 단계 0: 기반 정리
- [x] 초안 PR #1(NestJS 대화 루프)을 닫고, 동작과 테스트 케이스를 단계 3 명세로 옮김 ([specs/conversation-loop.md](./specs/conversation-loop.md))
- [x] `server/` Go 모듈 생성 (`go mod init`, `cmd/api/main.go`, 헬스체크 `GET /healthz`)
- [x] `app/` Flutter 저장소 생성 (`apps/elder_tablet`, `apps/caregiver`, `packages/*`)
- [x] GitHub Actions: Go(gofmt, `go vet`, `go test`), Flutter(`dart format`, `flutter analyze`, `flutter test`) — `sqlc diff`는 단계 1에서 추가
- [x] `docker-compose.yml`의 backend를 Go 서버 이미지로 교체 (기존 NestJS는 `legacy` 프로필로 유지)
- [x] 기존 `apps/` placeholder(Flutter·React Native 예정 메모) 삭제, 내용은 `app/`과 architecture.md로 이동

### 단계 1: 계약 확정
- [ ] 기존 API 명세 초안을 `server/api/openapi.yaml`로 옮기고 말동무·0번 문장 관련 엔드포인트 추가
- [ ] Prisma 스키마를 `server/db/migrations/00001_init.sql`로 옮기며 architecture.md 5절의 변경 반영
- [ ] WebSocket 이벤트를 JSON 스키마로 정의 (`server/api/ws-events.md`)
- [ ] 코드 생성 스크립트 (`make generate`): sqlc, oapi-codegen, Dart 클라이언트

### 단계 2: 서버 뼈대
- [ ] 설정(`caarlos0/env`), 로깅(`slog`), pgx 풀, 서버 시작 시 goose 마이그레이션
- [ ] 방문 목록·위치 수신 API, ETA 인터페이스(초기엔 가짜 구현)
- [ ] 조건부 `UPDATE`로 방문 상태 전이, ETA 기준 도달 시 세션 생성 (동시성 테스트 포함)
- [ ] 기기 토큰 인증 (태블릿), 조무사 임시 로그인
- [ ] `testcontainers-go`로 통합 테스트

### 단계 3: 대화 엔진 (텍스트)
- [ ] 규칙 필터와 위급 이벤트 (규칙에 `rule_id` 부여)
- [ ] 0번 문장 분류기와 `opener_clip` 시드 데이터
- [ ] Claude 스트리밍 호출, 문장 분할기, 대체 문장, 응답 시간 제한
- [ ] `flag_concern` 도구 처리 (`source=LLM`)
- [ ] 프롬프트 템플릿 파일과 버전 기록, 프롬프트 캐싱
- [ ] `/ws/elder` 핸들러 (텍스트 모드), 개발용 CLI 클라이언트
- [ ] PR #1에 있던 테스트 케이스를 Go 테스트로 옮김 ([specs/conversation-loop.md](./specs/conversation-loop.md) 4절)

### 단계 4: 태블릿 앱
- [ ] WebSocket 연결·재연결, 세션 화면 3개(대기, 대화, 도착 임박)
- [ ] 재생 큐: 0번 → 1번 → 2번, 끼어들기 시 취소
- [ ] 0번 문장 음성 캐시 내려받기
- [ ] 녹음·발화 구간 감지, 서버 STT·TTS(Clova) 연동
- [ ] 키오스크(Lock Task) 플랫폼 채널, Foreground Service
- [ ] 측정: 발화 종료 → 0번 재생 시작, 0번 종료 → 1번 재생 시작

### 단계 5: 조무사 앱과 비동기 작업
- [ ] 방문 목록, 지도·ETA 화면, 10초 간격 위치 전송
- [ ] FCM 위급 알림, 알림 확인(`ack`)
- [ ] River: 브리핑 생성(Haiku, JSON 스키마), 키워드 추출·감쇠 갱신
- [ ] 실제 ETA API(카카오모빌리티 또는 티맵) 연결

### 단계 6: 말동무 모드
- [ ] `mode`, `started_by`, `companion_schedule`, `daily_digest`
- [ ] 버튼으로 시작, 무응답·취침 시간 종료, 보호자 지정 시각 안부 대화
- [ ] 보호자 위급 알림, 하루 요약 발송
- [ ] 히스토리 요약(최근 약 10턴 + Haiku 요약), 어르신별 하루 토큰 한도

### 단계 7: 파일럿 준비
- [ ] 위급 키워드 목록 요양·의료 전문가 검토, 과소·과대 탐지 점검
- [ ] 대화·음성 보관 기간과 암호화 결정
- [ ] 배포 환경(NCP 또는 AWS 서울), 비밀값 관리, 백업
- [ ] 모니터링: 응답 지연, Claude 오류율, 위급 이벤트 수, 일일 토큰 사용량

## 3. 작업 방식

- **브랜치와 PR**: 기능 하나에 PR 하나. 한 PR은 리뷰 가능한 크기로 유지하고, 단계 번호를 제목에 붙입니다 (예: `[단계 2] 방문 상태 전이`).
- **머지 조건**: CI 통과, 새 로직에는 테스트, 계약(`openapi.yaml`, 마이그레이션, WebSocket 이벤트)을 바꾸면 생성 코드도 같은 PR에서 갱신.
- **마이그레이션**: 이미 머지된 마이그레이션 파일은 고치지 않고 새 파일을 추가합니다.
- **결정 기록**: 설계를 바꾸는 결정은 [architecture.md](./architecture.md) 7절 표에 날짜와 함께 추가합니다.
- **비밀값**: API 키는 `.env`(커밋 금지)와 배포 환경의 비밀값 저장소에만 둡니다. 앱에는 Claude·Clova 키를 넣지 않습니다.
- **안전 기능 우선**: 위급 필터·알림에 영향을 주는 변경은 테스트 케이스(감지돼야 하는 문장, 감지되면 안 되는 문장)를 함께 추가합니다.
