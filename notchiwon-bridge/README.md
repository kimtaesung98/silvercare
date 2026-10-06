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
│   ├── cmd/elder-cli/        # 개발용 텍스트 대화 클라이언트 (/ws/elder)
│   ├── api/                  # 계약: openapi.yaml, ws-events.md (+ JSON 스키마)
│   ├── db/                   # goose 마이그레이션, sqlc 쿼리
│   └── internal/             # session(대화 엔진·WebSocket), escalation, opener, llm, visit, httpapi, db·apigen(생성 코드) …
├── app/                      # Flutter (pub workspace)
│   ├── apps/elder_tablet/    # 어르신 태블릿 앱
│   ├── apps/caregiver/       # 조무사 앱
│   ├── apps/guardian/        # 보호자 앱
│   └── packages/             # api_client, voice, ui 공유 패키지
├── docs/                     # 아키텍처, 개발 과정
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
실제 계정은 `go run ./cmd/admin set-caregiver-login -caregiver <id> -login <아이디>`(비밀번호는 표준 입력), 보호자는 `go run ./cmd/admin set-guardian-login -guardian <id> -login <아이디>`, 태블릿은 `go run ./cmd/admin create-tablet -elder <id>`로 발급합니다.
ETA는 `KAKAO_MOBILITY_API_KEY`가 있으면 카카오모빌리티 길찾기로, 없으면 방문 예정 시각까지 남은 분으로 셉니다. ETA가 15분(`SESSION_TRIGGER_ETA_MINUTES`) 이내가 되면 픽업 대기 세션이 시작되고, 도착 처리하면 브리핑이 비동기로 만들어집니다.

### 텍스트로 대화해 보기

```bash
export ANTHROPIC_API_KEY=...                 # 없으면 매 턴 대체 문장으로 끝납니다
make run &
go run ./cmd/elder-cli -server http://localhost:8000 -token <seed-demo가 출력한 태블릿 토큰>
```

`/start`로 말동무 세션을 시작한 뒤 어르신 말을 입력하면 0번 문장(캐시 음성 자리), Claude가 이어 쓴 문장, 턴 결과가 순서대로 찍힙니다.
`/barge`는 끼어들기, `/end`는 그만하기, `/quit`은 종료입니다. "다리가 너무 아파"처럼 위급 규칙에 걸리는 말은 Claude를 부르지 않고 안심 문장으로 답합니다.

### 계약을 바꿀 때

`server/api/openapi.yaml`, `server/db/migrations/`, `server/db/queries/`를 바꾸면 `server/`에서 `make generate`를 돌려 생성 코드를 같은 PR에 커밋합니다.
Go 코드(sqlc, oapi-codegen)와 Dart 클라이언트(`app/packages/api_client/lib/src/generated`)를 만들며, Java 17+와 Flutter가 필요합니다.

전체 스택을 Docker로 실행하려면:

```bash
docker compose up --build server
```

## 앱 실행

```bash
cd app
flutter pub get                       # workspace 전체 한 번에

# 어르신 태블릿: 서버 주소와 기기 토큰을 빌드할 때 넣습니다
# (토큰은 POST /devices/{deviceId}/token으로 발급)
cd apps/elder_tablet && flutter run \
  --dart-define=SERVER_URL=http://10.0.2.2:8000 \
  --dart-define=DEVICE_TOKEN=...

# 조무사 앱: Firebase 값은 Firebase 콘솔 > 프로젝트 설정 > Android 앱(kr.notchiwon.caregiver)에서.
# 비워 두면 푸시 없이 동작하고 위급 알림은 홈 화면 목록으로만 봅니다.
cd apps/caregiver && flutter run \
  --dart-define=SERVER_URL=http://10.0.2.2:8000 \
  --dart-define=FIREBASE_API_KEY=... \
  --dart-define=FIREBASE_APP_ID=... \
  --dart-define=FIREBASE_SENDER_ID=... \
  --dart-define=FIREBASE_PROJECT_ID=...

# 보호자 앱: Firebase 값은 Android 앱(kr.notchiwon.guardian)에서. 값이 없으면
# 푸시 없이 동작하고 위급 알림은 홈 화면 목록으로만 봅니다.
cd apps/guardian && flutter run \
  --dart-define=SERVER_URL=http://10.0.2.2:8000 \
  --dart-define=FIREBASE_API_KEY=... \
  --dart-define=FIREBASE_APP_ID=... \
  --dart-define=FIREBASE_SENDER_ID=... \
  --dart-define=FIREBASE_PROJECT_ID=...

# 검사 (app/ 에서)
dart format . && flutter analyze
for pkg in apps/elder_tablet apps/caregiver apps/guardian packages/api_client packages/voice packages/ui; do
  (cd "$pkg" && flutter test)
done
```

어르신 태블릿은 켜면 바로 대화 화면이 뜨도록 키오스크(Lock Task)로 돌립니다.
잠금은 태블릿을 기기 소유자(Device Owner)로 등록했을 때만 걸리고, 등록하지 않은
기기에서는 보통 앱처럼 동작합니다. 0번 문장 음성은 서버에서 미리 합성해 두세요:

```bash
cd server && go run ./cmd/admin synth-openers
```

### 픽업 흐름 처음부터 끝까지 시험하기

1. `.env`에 `ANTHROPIC_API_KEY`, `NAVER_CLOVA_CLIENT_ID`/`SECRET`, `KAKAO_MOBILITY_API_KEY`, `FCM_CREDENTIALS_FILE`(Firebase 서비스 계정 JSON 경로)을 넣고 서버를 띄웁니다. 빠진 키가 있으면 시작 로그에 무엇이 대신 동작하는지 경고가 나옵니다.
2. `go run ./cmd/admin seed-demo`로 조무사 계정·30분 뒤 방문·태블릿 토큰을 받고, `go run ./cmd/admin synth-openers`로 0번 문장 음성을 만듭니다. 데모 어르신 댁은 서울 종로구 세종대로 175입니다(실제 주소는 DB `elder.home_*`에).
3. 태블릿(또는 에뮬레이터)에 어르신 앱을, 휴대폰에 조무사 앱을 설치합니다. 실제 기기에서는 `SERVER_URL`을 개발 PC의 사설 IP로 넣습니다.
4. 조무사 앱에서 로그인 → 방문 → "출발하기". ETA가 15분 이내면 태블릿에서 대화가 시작됩니다(가까이 있으면 `SESSION_TRIGGER_ETA_MINUTES`를 크게).
5. 태블릿에 "다리가 너무 아파"라고 말하면 휴대폰에 위급 알림이 옵니다. "도착했어요"를 누르면 몇 초 뒤 브리핑 카드가 뜹니다.

### 말동무와 보호자 앱 시험하기

1. 위와 같이 서버를 띄우고 `go run ./cmd/admin seed-demo`로 보호자 계정까지 받습니다 (`guardian login:` 줄).
2. 두 번째 휴대폰(또는 에뮬레이터)에 보호자 앱을 설치하고 그 계정으로 로그인합니다. 푸시를 받으려면 Firebase 값을 넣어야 합니다.
3. 보호자 앱 > 어르신 > 설정에서 안부 전화 시각을 **1~2분 뒤**로, 취침 시간을 조금 뒤로 정합니다. 1분마다 도는 틱이 그 시각에 태블릿에서 대화를 시작합니다 (태블릿이 연결되어 있어야 합니다).
4. 태블릿에서 "말동무" 버튼으로 직접 시작해도 됩니다. "가슴이 아파"라고 말하면 보호자 휴대폰에 위급 알림이 오고, "이제 그만할게"라고 하면 인사말과 함께 대화가 끝납니다.
5. 취침 시간이 시작되면(또는 `COMPANION_DIGEST_AT`에) 보호자 휴대폰에 그날의 하루 소식이 옵니다. 기다리지 않고 보려면 설정의 취침 시작을 2분 뒤로 옮기세요.

## CI

PR마다 `.github/workflows/server.yml`(gofmt, go vet, Postgres를 띄운 go test), `app.yml`(dart format, flutter analyze, flutter test), `contracts.yml`(생성 코드가 계약과 같은지)이 돕니다.

## 다음 단계

[docs/development-process.md](docs/development-process.md)의 단계 7(파일럿 준비: 위급 키워드 전문가 검토, 보관 기간과 암호화, 배포 환경, 모니터링)입니다.

## 참고 문서
- 기획서: `노치원_AI_브릿지_시스템_기획서.md`
- 아키텍처 & 데이터 모델: `노치원_AI_브릿지_아키텍처_데이터모델.md`
