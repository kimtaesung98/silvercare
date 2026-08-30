# 노치원 픽업 브릿지 시스템 (Notchiwon Pickup Bridge)

조무사 도착 전 병목 구간 동안 AI 대화로 노인의 정서적 안정을 돕고,
조무사에게는 실시간 브리핑을 제공하는 시스템입니다.

## 레포 구조

```
notchiwon-bridge/
├── backend/                # NestJS 기반 API 서버 (ETA 엔진, 대화 오케스트레이터, 키워드 엔진, 브리핑 엔진)
│   ├── prisma/              # DB 스키마 및 마이그레이션
│   └── src/
│       ├── modules/
│       │   ├── visit/        # 방문 스케줄, 위치 수신, ETA 계산
│       │   ├── session/      # 대화 세션 오케스트레이션, Claude API 연동
│       │   ├── keyword/      # 관심사 키워드 추출·가중치 갱신
│       │   ├── briefing/     # 조무사 브리핑 리포트 생성
│       │   └── escalation/   # 위급 상황 감지·알림
│       └── main.ts
├── apps/
│   ├── elder-tablet/         # 노인용 태블릿 앱 (Flutter 예정, 현재 placeholder)
│   └── caregiver-app/        # 조무사용 모바일 앱 (React Native 예정, 현재 placeholder)
├── infra/                    # 배포/인프라 관련 설정 (추후 k8s, CI/CD 등)
├── docker-compose.yml        # 로컬 개발 환경 (Postgres, Redis, backend)
└── .env.example
```

## 로컬 개발 환경 실행

### 1. 사전 준비
- Docker & Docker Compose 설치
- Node.js 20+ (백엔드 로컬 실행 시)
- `.env.example`을 `.env`로 복사 후 API 키 등 값 채우기

```bash
cp .env.example .env
```

### 2. 인프라(Postgres, Redis) 기동

```bash
docker compose up -d postgres redis
```

### 3. 백엔드 의존성 설치 및 DB 마이그레이션

```bash
cd backend
npm install
npx prisma migrate dev --name init
```

### 4. 백엔드 개발 서버 실행

```bash
npm run start:dev
```

기본적으로 `http://localhost:3000` 에서 API 서버가 뜹니다.

### 5. 전체 스택을 Docker로 한 번에 실행 (선택)

```bash
docker compose up --build
```

## 다음 단계 (개발 체크리스트)

- [ ] `backend/prisma/schema.prisma` 검토 및 실제 요구사항에 맞게 필드 보완
- [ ] `visit` 모듈: 조무사 앱으로부터 위치 수신 → ETA 계산 로직 구현
- [ ] `session` 모듈: Claude API 연동, 시스템 프롬프트(공감형 톤 + 안전장치) 작성
- [ ] `keyword` 모듈: Kiwi 형태소 분석기 연동 (Python 서브 서비스 또는 Node 바인딩 검토)
- [ ] `escalation` 모듈: 위급 키워드 필터 규칙 목록 정의 (도메인 전문가 검토 필요)
- [ ] 노인용 태블릿 앱: STT/TTS(Naver Clova) 연동 프로토타입
- [ ] 조무사용 앱: 지도 SDK(카카오모빌리티/티맵) 연동, 위치 전송 주기 설정

## 참고 문서
- 기획서: `노치원_AI_브릿지_시스템_기획서.md`
- 아키텍처 & 데이터 모델: `노치원_AI_브릿지_아키텍처_데이터모델.md`
