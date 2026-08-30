# 조무사용 모바일 앱 (placeholder)

## 예정 스택
- React Native
- 지도 SDK: 카카오모빌리티 또는 티맵(Tmap)
- 위치 전송: 5~10초 주기 REST 호출 (`POST /visits/location`) 또는 WebSocket

## 핵심 화면
1. 오늘의 방문 목록 (순서, 예정 시각, 상태)
2. 지도 + 실시간 ETA 뷰
3. 브리핑 카드 (도착 임박 시 자동 표시: 요약, 오늘의 화제, 특이사항 플래그)
4. 위급 알림 수신 화면 (푸시 + 인앱 배너)

## TODO
- [ ] React Native 프로젝트 초기화
- [ ] 지도 SDK 연동 및 위치 전송 주기 테스트 (배터리 소모 고려)
- [ ] 브리핑 카드 UI ↔ `GET /briefings/:sessionId` 연동
