# 노인용 태블릿 앱 (placeholder)

## 예정 스택
- Flutter (Android 우선 타겟, 키오스크 모드 설정)
- STT/TTS: Naver Clova Speech / Clova Voice
- 백엔드와의 통신: WebSocket (실시간 세션 트리거 수신) + REST (발화 전송)

## 핵심 화면
1. 대기 화면 (평상시 — 조무사 미접근 상태)
2. 대화 세션 화면 (AI 발화 표시/재생, 노인 음성 입력 상시 대기)
3. "조무사 도착 임박" 안내 화면 (예측 가능성 제공용 카운트다운/안내 문구)

## 설계 원칙
- 터치보다 음성 우선, 큰 글씨/고대비, 웨이크워드 없는 상시 음성 인터랙션
- 화면 전환은 최소화 (인지 부담 감소)

## TODO
- [ ] Flutter 프로젝트 초기화 (`flutter create .`)
- [ ] 백엔드 WebSocket 이벤트 스펙 확정 후 클라이언트 연동
- [ ] Clova STT/TTS SDK 연동 프로토타입
