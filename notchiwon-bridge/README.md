# 노치원 픽업 브릿지 시스템 (Notchiwon Pickup Bridge)

조무사 도착 전 병목 구간 동안 AI 대화로 노인의 정서적 안정을 돕고,
조무사에게는 실시간 브리핑을 제공하는 시스템입니다. 하원 후에도 말동무 대화를 이어갑니다.

구조와 설계 근거는 [docs/architecture.md](docs/architecture.md), 진행 순서는 [docs/development-process.md](docs/development-process.md)를 참고하세요.

## 레포 구조

```
notchiwon-bridge/
├── server/                   # Go API·WebSocket 서버
│   ├── cmd/api/              # 실행 진입점
│   ├── cmd/admin/            # 태블릿 토큰·조무사 계정 발급, 데모 데이터
│   ├── api/                  # 계약: openapi.yaml, ws-events.md (+ JSON 스키마)
│   ├── db/                   # goose 마이그레이션, sqlc 쿼리
│   └── internal/             # config, httpapi, db·apigen(생성 코드), 모듈은 단계별로 추가
├── app/                      # Flutter (pub workspace)
│   ├── apps/elder_tablet/    # 어르신 태블릿 앱
│   ├── apps/caregiver/       # 조무사 앱
│   └── packages/             # api_client, voice, ui 공유 패키지
├── docs/                     # 아키텍처, 개발 과정
├── backend/                  # 기존 NestJS 서버 (전환 기간에만 유지, 단계 3 이후 삭제)
├── docker-compose.yml        # postgres, redis, server
└── .env.example
```

## 사전 준비

- Docker & Docker Compose
- Go 1.25+
- Flutter 3.27+ (앱 작업 시)
- `.env.example`을 `.env`로 복사 후 값 채우기

```bash
cp .env.example .env
```

## 서버 실행

```bash
# 인프라 기동
docker compose up -d postgres redis

# 개발 실행
cd server
make run          # http://localhost:8000
curl localhost:8000/healthz

# 검사
make fmt vet test

# DB 테스트까지 (임시 DB를 만들고 지웁니다)
TEST_DATABASE_URL="postgres://notchiwon:notchiwon_dev_pw@localhost:5432/notchiwon_bridge?sslmode=disable" make test

# 마이그레이션 적용
DATABASE_URL="postgres://notchiwon:notchiwon_dev_pw@localhost:5432/notchiwon_bridge?sslmode=disable" make migrate-up
```

### 로컬에서 API 써 보기

```bash
cd server
export DATABASE_URL="postgres://notchiwon:notchiwon_dev_pw@localhost:5432/notchiwon_bridge?sslmode=disable"
export AUTH_SECRET="$(openssl rand -base64 48)"
make run &                                   # 시작하면서 마이그레이션 적용
go run ./cmd/admin seed-demo                 # 데모 센터·어르신·조무사 계정·30분 뒤 방문·태블릿 토큰 출력
```

조무사 앱은 `POST /auth/caregiver/login`으로 받은 토큰, 태블릿은 기기 토큰을 `Authorization: Bearer`로 보냅니다.
실제 계정은 `go run ./cmd/admin set-caregiver-login -caregiver <id> -login <아이디>`(비밀번호는 표준 입력), 태블릿은 `go run ./cmd/admin create-tablet -elder <id>`로 발급합니다.
지금 ETA는 가짜(방문 예정 시각까지 남은 분)라서 예정 15분 전부터 위치를 보내면 픽업 대기 세션이 시작됩니다.

### 계약을 바꿀 때

`server/api/openapi.yaml`, `server/db/migrations/`, `server/db/queries/`를 바꾸면 `server/`에서 `make generate`를 돌려 생성 코드를 같은 PR에 커밋합니다.
Go 코드(sqlc, oapi-codegen)와 Dart 클라이언트(`app/packages/api_client/lib/src/generated`)를 만들며, Java 17+와 Flutter가 필요합니다.

전체 스택을 Docker로 실행하려면:

```bash
docker compose up --build server
```

전환 기간 동안 기존 NestJS 서버가 필요하면 `docker compose --profile legacy up backend`로 띄웁니다.

## 앱 실행

```bash
cd app
flutter pub get                       # workspace 전체 한 번에

cd apps/elder_tablet && flutter run    # 어르신 태블릿
cd apps/caregiver && flutter run       # 조무사 앱

# 검사 (app/ 에서)
dart format . && flutter analyze
cd apps/elder_tablet && flutter test
```

## CI

PR마다 `.github/workflows/server.yml`(gofmt, go vet, Postgres를 띄운 go test), `app.yml`(dart format, flutter analyze, flutter test), `contracts.yml`(생성 코드가 계약과 같은지)이 돕니다.

## 다음 단계

[docs/development-process.md](docs/development-process.md)의 단계 3(대화 엔진, 텍스트)입니다.

## 참고 문서
- 기획서: `노치원_AI_브릿지_시스템_기획서.md`
- 아키텍처 & 데이터 모델: `노치원_AI_브릿지_아키텍처_데이터모델.md`
