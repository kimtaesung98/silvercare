# 명세 체계

> 이 폴더의 목표: 명세만 읽은 어느 에이전트가 만들어도 **같은 결과**가 나오게 하는 것.
> 처음 정한 날: 2026-10-20. 바탕은 프로젝트 공유 폴더 `연구/명세서-작업-요구사항.md`의 프로토 판입니다.

## 1. 명세의 네 층

```
 0층  방향           ../DIRECTION.md                왜 만드나, 원칙          (주인이 정함)
 1층  요구사항       requirements.md               누가 무엇을 원하나        UR-… 사용자 / DR-… 개발자
 2층  중위 명세      mid/<이름>.md                  부품 묶음 하나의 약속     예: mid/voice.md
 3층  하위 명세      low/<묶음>/<이름>.md           부품 하나를 만들 만큼     예: low/voice/recorder.md
```

- **위층이 아래층을 낳습니다.** 1층의 요구사항 한 줄은 하나 이상의 2층 명세에 배정되고, 2층의 요구사항은 3층 명세로 나뉩니다. 위층에 없는 것을 아래층에서 새로 만들지 않습니다. 필요하면 위층부터 고칩니다.
- **번호가 길을 남깁니다.** 아래층 요구사항은 자기를 낳은 위층 번호를 `근거` 칸에 적습니다. 그래서 `UR-E-03 → VOICE-04 → REC-07`처럼 어르신의 바람에서 녹음기 한 줄까지 따라갈 수 있습니다.
- 2층은 DIRECTION.md 4절의 네 층(바탕 · 참가자 · 모듈 · 서비스)과 앱, 공통 기반으로 나눕니다. 목록과 진행 상태는 아래 4절에 있습니다.

## 2. 쓰는 규칙 (모든 층 공통)

1. **정해진 틀**: 2층은 [templates/mid.md](./templates/mid.md), 3층 부품은 [templates/low.md](./templates/low.md), 3층 화면은 [templates/screen.md](./templates/screen.md)의 절 순서와 제목을 그대로 씁니다. 쓸 것이 없는 절은 지우지 않고 "해당 없음"이라고 적습니다.
2. **한 줄에 요구사항 하나**: 번호, 강도, 내용, 근거, 수락 기준 번호를 표 한 줄에 씁니다.
3. **강도는 셋 중 하나**: **반드시**(어기면 틀림), **되도록**(어기려면 이유를 적음), **해도 됨**(만드는 쪽이 고름. 고른 것을 기록).
4. **숫자는 단위까지**: "짧게", "적당히", "빠르게"는 쓰지 않습니다. 모르면 열린 질문으로 둡니다.
5. **수락 기준은 Given/When/Then**: "~일 때(상황), ~하면(행동), ~이다(결과)". 요구사항마다 하나 이상이고, 가짜 부품이나 녹음 파일로 돌릴 수 있게 씁니다.
6. **코드는 쓰지 않습니다**: 인터페이스는 이름 · 입력 · 출력 · 오류를 표로만 적습니다. 의사코드, 코드 조각, 특정 라이브러리 함수 호출은 쓰지 않습니다. 라이브러리 이름은 "플랫폼" 절에 고른 이유와 함께 적을 수 있습니다.
7. **가정과 열린 질문**: 만드는 쪽이 스스로 정한 값은 `가정`으로 표시하고 근거(이전 결정 번호, 파일)를 답니다. 정할 수 없는 것은 `열린 질문`으로 남기고 혼자 정하지 않습니다. 열린 질문은 주인이 답하면 요구사항이나 가정으로 옮깁니다.
8. **화면 연결**: 화면이 있는 명세는 피그마 프레임 링크를 겁니다. 피그마에 없는 화면은 만들지 않고 열린 질문에 "필요한 화면"으로 적습니다 (DIRECTION.md 5절).
9. **용어는 [glossary.md](./glossary.md)의 뜻으로만** 씁니다. 새 말이 필요하면 거기에 먼저 더합니다.

## 3. 명세가 끝났다고 보는 기준

한 명세는 아래를 모두 넘으면 `확정`입니다.

1. 틀의 모든 절이 채워졌거나 "해당 없음"이다.
2. 모든 요구사항에 근거와 수락 기준이 있다.
3. 위층 요구사항 중 이 명세에 배정된 것이 모두 아래로 이어졌다 ([requirements.md](./requirements.md) 4절 추적표).
4. **쌍둥이 시험**을 통과했다: 서로 모르는 에이전트 둘이 이 명세만 보고 실험 브랜치에 따로 만든 뒤, 수락 기준을 양쪽에 돌려 결과가 같다. 결과가 갈린 곳은 명세의 빈칸으로 보고 메운 뒤 다시 시험합니다. 실험 코드는 머지하지 않습니다.

상태: `빈칸`(목록에만 있음) → `초안` → `검토`(주인 확인 중) → `확정`.

## 4. 명세 지도

| 2층 중위 명세 | 층 | 3층 하위 명세 | 상태 |
| --- | --- | --- | --- |
| `mid/chat-core.md` 대화 바탕 | 바탕 | `low/chat/room.md` 방과 참가자, `low/chat/message.md` 메시지와 표시, `low/chat/delivery.md` 실시간 전달, `low/chat/module-slot.md` 모듈 자리 | 빈칸 |
| `mid/companion.md` 말동무 AI | 참가자 | `low/companion/turn.md` 한 턴 만들기(0번 문장 포함), `low/companion/brain.md` 생각하는 부분(Claude와 대체 동작), `low/companion/limits.md` 시간·토큰 한도 | 빈칸 |
| `mid/safety.md` 위급 필터 | 모듈 | `low/safety/rules.md` 규칙 필터, `low/safety/guard.md` 안전 지킴이와 안심 문장, `low/safety/escalation.md` 사람에게 알리기 | 빈칸 |
| `mid/voice.md` 음성 | 모듈 | `low/voice/recorder.md` 녹음기, `low/voice/segmenter.md` 말 구간 자르기, `low/voice/stt.md` 말을 글로, `low/voice/tts.md` 글을 말로, `low/voice/playback.md` 재생 순서 | 빈칸 |
| `mid/summary.md` 요약 | 모듈 | `low/summary/briefing.md` 조무사 브리핑, `low/summary/daily-news.md` 하루 소식 | 빈칸 |
| `mid/pickup.md` 픽업 대기 | 서비스 | `low/pickup/trip.md` 출발과 위치, `low/pickup/eta.md` 남은 시간, `low/pickup/room-rules.md` 방 열고 닫기 | 빈칸 |
| `mid/ui-system.md` 공통 화면 규칙 | 앱 공통 | `low/ui/tokens.md` 색·글씨·간격 변수, `low/ui/components.md` 컴포넌트 짝 표, `low/ui/states.md` 다섯 가지 모습, `low/ui/copy.md` 문구 규칙과 공통 문구 | 빈칸 |
| `mid/elder-tablet.md` 어르신 태블릿 앱 | 앱 | 화면: `low/elder-tablet/screen-idle.md` 대기, `screen-talk.md` 대화, `screen-arriving.md` 도착 임박, `screen-after-alert.md` 위급 뒤, `screen-setup.md` 직원용 처음 설정 · 부품: `navigation.md` 화면 지도, `kiosk.md` 화면 고정과 계속 켜 두기 | 빈칸 |
| `mid/caregiver-app.md` 조무사 앱 | 앱 | 화면: `low/caregiver-app/screen-login.md` 로그인, `screen-today.md` 오늘 방문, `screen-trip.md` 이동 중, `screen-briefing.md` 브리핑, `screen-alert.md` 위급, `screen-me.md` 내 정보와 설정 · 부품: `navigation.md` 화면 지도, `alerts.md` 위급 알림 받기 | 빈칸 |
| `mid/guardian-app.md` 보호자 앱 | 앱 | 화면: `low/guardian-app/screen-login.md`, `screen-home.md`, `screen-daily.md` 하루 소식, `screen-alert.md`, `screen-settings.md` 안부 시각, `screen-me.md` · 부품: `navigation.md` | 빈칸 (나중) |
| `mid/platform.md` 공통 기반 | 기반 | `low/platform/contracts.md` 메시지 약속, `low/platform/storage.md` 보관과 개인정보, `low/platform/auth.md` 기기·사람 확인, `low/platform/ops.md` 실행·키·검사 | 빈칸 |

화면 명세는 피그마 프레임이 먼저입니다. 프레임이 없는 화면은 명세 상태가 `빈칸`에서 넘어가지 않습니다.

처음 만들 순서 (추천): `requirements.md` 확정 → `mid/voice.md` → `low/voice/recorder.md`(첫 쌍둥이 시험) → `mid/chat-core.md` → `mid/safety.md` → 나머지.
녹음이 첫 대상인 이유: 주인이 처음 든 예이고, 이전 버전에서 가장 오래 막힌 곳(에뮬레이터 마이크)이라 명세가 빈칸을 가장 많이 드러냅니다.

## 5. 에이전트에게 줄 지시문

명세 한 편을 맡길 때 아래를 그대로 주고 `[ ]`만 바꿉니다.

```
silvercare 저장소의 명세 작업입니다. 코드는 쓰지 않습니다.

먼저 읽을 것: docs/DIRECTION.md, docs/specs/README.md, docs/specs/requirements.md,
docs/specs/glossary.md, 그리고 이 명세의 위층 명세 [mid/voice.md]
형식 참고(내용은 그대로 옮기지 않음): notchiwon-bridge/docs/handbook/specs/, notchiwon-bridge/docs/architecture.md 결정 표

할 일: [low/voice/recorder.md]를 [templates/low.md] 틀로 씁니다.
배정된 위층 요구사항: [VOICE-01 ~ VOICE-06] (requirements.md 4절 추적표)

README.md 2절 규칙 1~9를 지킵니다. 특히
- 위층에 없는 부품, DIRECTION.md 1~5절과 부딪히는 내용, 피그마에 없는 화면은 만들지 않고 열린 질문에 적습니다.
- 스스로 정한 값은 "가정"과 근거를 답니다.

끝나면: 명세 지도(README.md 4절)의 상태를 고치고, 추적표를 채우고, 초안 PR을 올리고,
열린 질문 목록을 주인에게 보여 줍니다. DIRECTION.md 7절에 한 줄 남깁니다.

끝났다고 볼 기준: README.md 3절의 1~3. (4번 쌍둥이 시험은 따로 맡깁니다.)
```

쌍둥이 시험을 맡길 때:

```
docs/specs/[low/voice/recorder.md] 만 읽고, 실험 브랜치 [spec-twin/recorder-a]에 그대로 만듭니다.
다른 실험 브랜치와 이전 버전 코드(notchiwon-bridge/)는 보지 않습니다.
명세에 없어서 스스로 정해야 했던 것을 모두 목록으로 남깁니다. 그 목록이 이 시험의 주된 결과입니다.
수락 기준을 모두 테스트로 만들어 돌리고 결과를 남깁니다. PR은 올리지 않습니다.
```
