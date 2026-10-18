# 진행 상황 (2026-10-11)

> 기준: master `8110f59`. 단계 정의는 [development-process.md](../development-process.md)입니다.

## 1. 한눈에

8단계 중 **단계 0~6의 개발은 끝났고**, 단계 7(파일럿 준비)의 첫 작업인 태블릿 음성을 마쳤습니다 (PR #13). 서버와 앱 세 개(어르신 태블릿, 조무사, 보호자)가 모두 있고, 외부 키가 없어도 전체 흐름이 대체 동작으로 돌아갑니다. 남은 것은 **실제 키와 실제 기기로 확인하는 주인 검토**, 그리고 배포·모니터링·전문가 검토입니다.

```mermaid
flowchart LR
  s0["0 기반"]:::done --> s1["1 계약"]:::review --> s2["2 서버 뼈대"]:::done --> s3["3 대화 엔진"]:::review --> s4["4 태블릿"]:::review --> s5["5 조무사 앱"]:::review --> s6["6 말동무"]:::review --> s7["7 파일럿 준비"]:::doing
  classDef done fill:#d3f9d8,stroke:#2b8a3e,color:#000
  classDef review fill:#fff3bf,stroke:#e67700,color:#000
  classDef doing fill:#d0ebff,stroke:#1971c2,color:#000
```

초록: 끝남 · 노랑: 개발은 끝, 주인 확인 남음 · 파랑: 진행 중

## 2. 단계별

| 단계 | 개발 | 남은 것 |
| --- | --- | --- |
| 0 기반 정리 | 끝 | 없음 |
| 1 계약 확정 (REST·DB·WebSocket·코드 생성) | 끝 | 주인 검토 |
| 2 서버 뼈대 (방문·위치·세션 트리거·기기 인증) | 끝 | 없음 |
| 3 대화 엔진 (규칙 필터·0번 문장·Claude 스트리밍) | 끝 | 실제 Claude 키로 대화 품질 확인 |
| 4 태블릿 앱 (화면·재생 큐·녹음·키오스크) | 끝 | 실제 태블릿 음성 대화 |
| 5 조무사 앱 (방문·지도·위급 알림·브리핑) | 끝 | 기기 두 대로 픽업 흐름 처음부터 끝까지 |
| 6 말동무 모드 (버튼·안부·하루 소식·보호자 앱) | 끝 | 보호자 앱으로 알림 받아 보기 |
| 7 파일럿 준비 | 진행 중 | 아래 5절 |

## 3. 최근 작업

| 날짜 | 작업 | 내용 |
| --- | --- | --- |
| 2026-10-11 | PR #13 태블릿 음성 | 기기 음성 인식이 어르신 말을 글자로 보내고, AI 답은 기기 TTS로 읽음. Clova 키를 넣으면 `scripts/dev.sh`가 자동으로 Clova 경로를 씀 |
| 2026-10-11 | 에뮬레이터 실행 | `TABLET_DEVICE="emulator-5558"`로 태블릿 앱 실행까지 확인. 음성 대화는 아직 |
| 2026-10-11 | PR #12 개발 스크립트 | `.env` 하나로 서버·데모 데이터·앱 세 개를 실행하는 `scripts/dev.sh` |
| 2026-10-05~07 | PR #2~#11 | 단계 0~6 서버·앱 구현 |

## 4. 키별 동작

| 키 | 없을 때 | 넣으면 |
| --- | --- | --- |
| `ANTHROPIC_API_KEY` | 매 턴 대체 문장, 브리핑·하루 소식은 어르신 말 인용 | 실제 대화와 요약 |
| `NAVER_CLOVA_CLIENT_ID/SECRET` | 태블릿 내장 음성 인식 | Clova 인식 (`TTS_PROVIDER=clova`면 목소리도) |
| `TTS_PROVIDER=polly` + AWS 키 | 태블릿 내장 TTS | Amazon Polly 목소리 (넣은 뒤 `scripts/dev.sh openers`) |
| `KAKAO_MOBILITY_API_KEY` | 예정 시각까지 남은 분을 ETA로 | 실제 도로 ETA |
| `FCM_CREDENTIALS_FILE` + 앱 Firebase 값 | 위급 알림은 앱 목록에만 (30초마다 새로 읽음) | 휴대폰 푸시 |

## 5. 다음 할 일

- [ ] emulator-5558에서 말동무 버튼을 눌러 음성 대화 확인 ([CR-14](./code-review.md#cr-14))
- [ ] Clova CSR 키와 AWS(Polly) 키를 `.env`에 넣고 `scripts/dev.sh openers`
- [ ] Claude 키로 대화 품질 확인 (단계 3 검토)
- [ ] 안전 경로 고치기: [CR-01](./code-review.md#cr-01) 규칙 필터, [CR-02](./code-review.md#cr-02) 잘린 답의 `flag_concern`
- [ ] [CR-03](./code-review.md#cr-03) 긴 대화 요약
- [ ] 배포: NCP 또는 AWS 서울, 비밀값 관리, 백업, HTTPS/WSS
- [ ] 모니터링: 응답 지연(`turn latency` 로그), Claude 오류율, 위급 이벤트 수, 일일 토큰
- [ ] 태블릿 기기 등록 흐름 ([CR-11](./code-review.md#cr-11))

## 6. 정해야 할 것

| 항목 | 선택지 | 관련 |
| --- | --- | --- |
| 대화 원문·음성 보관 기간과 암호화 | 개인정보 검토 필요 | architecture.md 8절 |
| 배포 위치 | NCP / AWS 서울 | architecture.md 8절 |
| 조무사 앱 지도 타일 | 카카오 / 티맵 | 결정 31 |
| 로그인 방식 | 전화번호 인증 / 센터 발급 계정(현재) | architecture.md 8절 |
| 위급 키워드 목록 | 요양·의료 전문가 검토 | CR-01 |
| 조무사에게 못 보낸 위급 알림 | 보호자 / 센터로 넘김 | CR-04 |
| 하루 소식 대상 | 설정 저장한 어르신 / 대화가 있었던 어르신 | CR-08 |
| 말동무 설정 권한 | 보호자만 / 조무사도 | CR-12 |
