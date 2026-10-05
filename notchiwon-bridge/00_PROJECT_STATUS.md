# 프로젝트 현재 진행 상황 정리 (Status & Recovery)
> 기준 시점: 2026-08-30. 컴퓨터 포맷 이후 재개를 위한 문서입니다.

---

## 1. 지금까지 진행된 것 (요약)

| 단계 | 산출물 | 파일명 |
|---|---|---|
| 1. 문제 정의 & 기획 | 병목 구간 문제 정의, 시스템 구성요소, MVP 범위, 요구사항 정리 | `01_기획서.md` |
| 2. 아키텍처 & 데이터 모델 | 전체 시스템 구조도, 이벤트 흐름, 8개 테이블 스키마 설계 | `02_아키텍처_데이터모델.md` |
| 3. 프로젝트 스캐폴딩 | NestJS 백엔드 골격 코드, Prisma 스키마(실제 DB 모델), Docker Compose, 앱 placeholder | `03_backend_scaffold/` (아래 전체 트리 참고) |

### 핵심 설계 결정 사항 (다시 읽지 않아도 되도록 요약)
- **관심사 태그**: 사전 등록형이 아니라 대화에서 추출된 키워드를 **시간 감쇠 가중치**(`score = score*0.9 + 오늘_언급수`)로 갱신 → `keyword_tag` 테이블 + `keyword.service.ts`에 구현됨.
- **AI 모델**: 실시간 대화는 Claude Sonnet급, 브리핑 요약은 Claude Haiku급(비용·속도 최적화).
- **안전장치**: 위급 상황(낙상/통증/자타해)은 LLM 단독 판단이 아니라 **규칙 기반 키워드 필터를 1차 방어선**으로 두고, 감지 시 조무사/센터에 즉시 에스컬레이션 (`escalation.service.ts`).
- **인프라**: PostgreSQL + Redis, 개인정보 이슈로 국내 리전(NCP 또는 AWS 서울) 고려.
- **지도/ETA**: 카카오모빌리티 또는 티맵 (국내 도로 상황 반영도가 구글맵보다 높음).
- **음성**: Naver Clova STT/TTS.

---

## 2. 전체 파일 트리 (현재 스캐폴딩 상태)

```
notchiwon-bridge/
├── README.md                          # 실행 방법 총정리
├── docker-compose.yml                 # postgres, redis, backend, adminer
├── .env.example                       # 필요한 API 키/환경변수 목록
├── .gitignore
├── docs_기획서.md
├── docs_아키텍처_데이터모델.md
├── backend/
│   ├── package.json
│   ├── tsconfig.json
│   ├── nest-cli.json
│   ├── Dockerfile
│   ├── prisma/
│   │   └── schema.prisma              # 8개 테이블 전체 정의
│   └── src/
│       ├── main.ts
│       ├── app.module.ts
│       ├── prisma/ (prisma.module.ts, prisma.service.ts)
│       └── modules/
│           ├── visit/       (controller, service, module) — ETA 트리거
│           ├── session/     (controller, service, module) — Claude API 연동
│           ├── keyword/     (service, module)              — 관심사 태그 로직
│           ├── briefing/    (service, module)               — 조무사 브리핑
│           └── escalation/  (service, module)                — 위급 감지 필터
└── apps/
    ├── elder-tablet/README.md          # Flutter 예정, 설계 원칙 명시
    └── caregiver-app/README.md         # React Native 예정
```

---

## 3. 컴퓨터 포맷 이후, 새로 세팅해야 할 로컬 개발 도구

아래는 이 프로젝트를 로컬에서 실행하기 위해 **새 컴퓨터에 반드시 설치해야 하는 것들**입니다.

| 도구 | 용도 | 확인 명령어 |
|---|---|---|
| **Git** | 버전 관리 | `git --version` |
| **Node.js 20 LTS** | 백엔드(NestJS) 실행 | `node --version` |
| **Docker Desktop** | Postgres/Redis 컨테이너 실행 | `docker --version` |
| **VS Code (또는 선호 에디터)** | 코드 편집 | - |
| (선택) **Flutter SDK** | 노인용 태블릿 앱 개발 시 | `flutter --version` |
| (선택) **React Native / Expo CLI** | 조무사 앱 개발 시 | `npx react-native --version` |

> Node.js는 [nodejs.org](https://nodejs.org)에서 LTS 버전 설치, Docker Desktop은 [docker.com](https://www.docker.com/products/docker-desktop/)에서 OS에 맞게 설치하시면 됩니다.

### 계정/키 관련 — 다시 발급받아야 할 수 있는 것들
포맷 전에 아래 항목을 발급받아 두셨다면 **다시 확인이 필요**합니다 (로컬에만 저장했다면 유실됐을 가능성):
- [ ] Anthropic API 키 (console.anthropic.com)
- [ ] 카카오모빌리티 또는 티맵 API 키
- [ ] Naver Clova STT/TTS 인증 정보
- [ ] Firebase 프로젝트(FCM) 설정
- [ ] Git 원격 저장소(GitHub 등) — **코드를 클라우드에 올려두지 않으셨다면, 이번 압축 파일이 유일한 백업본입니다.**

---

## 4. 새 컴퓨터에서 프로젝트 복원하는 순서

1. 첨부된 `notchiwon-bridge.zip` 압축 해제
2. **가장 먼저 할 일**: Git 저장소로 초기화해서 원격 저장소(GitHub 등)에 백업
   ```bash
   cd notchiwon-bridge
   git init
   git add .
   git commit -m "초기 스캐폴딩 복원"
   # 이후 GitHub 등에 원격 저장소 생성 후 push 권장 (재포맷 시 유실 방지)
   ```
3. `README.md`의 안내를 따라 `.env` 설정 → `docker compose up -d postgres redis` → `npm install` → `npx prisma migrate dev` → `npm run start:dev`

---

## 5. 다음 진행할 항목 (이전에 논의하던 것)

> 2026-10-05: 기술 스택을 Go 서버 + Flutter 앱으로 바꾸기로 했습니다. 새 구조는 [docs/architecture.md](docs/architecture.md), 진행 순서는 [docs/development-process.md](docs/development-process.md)를 따릅니다. 아래 NestJS 기준 내용은 전환 전 기록입니다.

- [ ] API 명세서 (엔드포인트 설계)
- [ ] 비기능 요구사항 (동시접속 규모, 응답 지연 허용치, 장애 폴백)
- [ ] 개발 일정 & 마일스톤

**강력히 권장**: 위 셋 중 무엇을 하든, **먼저 Git 원격 저장소부터 만들어 지금까지의 산출물을 push**해두시길 권합니다. 그래야 이런 유실 걱정 없이 이어서 작업하실 수 있습니다.
