# 어르신 태블릿 앱 (Flutter, Android 태블릿 먼저)

보호자가 켜 두고 떠나는 태블릿입니다. 화면 기준은 피그마 [실버케어 화면](https://www.figma.com/design/eOm90ve0gRyfLPEJHsByvt)의 어르신 태블릿 프레임이고, 표는 [docs/screens.md](../../../docs/screens.md)에 있습니다.

| 화면 | 피그마 | 코드 |
| --- | --- | --- |
| 보호자 켜 두기 | 11:71 | `lib/src/guardian_screen.dart` |
| 기다림 · 듣는 중 · 말하는 중 | 11:37 · 11:47 · 11:59 | `lib/src/elder_screen.dart`, 마이크 원은 `mic_indicator.dart` (컴포넌트 11:36) |

- 앱은 v2 서버 채팅에 **어르신 참가자**로 들어갑니다 (`chat_client.dart`). 위급 필터는 서버에서 말동무보다 먼저 봅니다.
- 듣기와 읽어 주기는 Android에 들어 있는 음성 인식과 TTS입니다 (`voice.dart`). 서버 음성 모듈이 생기면 `Voice` 자리에 바꿔 끼웁니다.
- 보호자 화면으로 돌아가는 법은 아직 정하는 중이라(PR #29 열린 질문 2) 임시로 **시계 3초 누르기**이고, `guardian_door.dart` 하나만 고치면 바뀝니다.
- 색은 피그마 `실버케어` 변수 모음과 같은 값을 `tokens.dart`에 둡니다. 화면은 피그마 프레임(1280×800) 크기로 그린 뒤 기기에 맞춥니다.

## 해 보기

```bash
# 1) 서버 (컴퓨터)
cd v2 && go run ./cmd/server

# 2) 앱 (Android 에뮬레이터: 10.0.2.2가 컴퓨터)
cd v2/apps/elder_tablet && flutter run
# 실기기: 같은 와이파이에서 컴퓨터 IP로
flutter run --dart-define=SERVER_URL=http://192.168.0.10:8080
```

1. 보호자 화면에서 세 가지 확인(충전·인터넷·마이크)을 보고, 어르신 이름을 적고 "켜 두고 나가기".
2. "안녕" 하고 말하면 원이 초록(듣는 중)이 됐다가 주황(말하는 중)으로 바뀌며 말동무가 소리로 대답합니다.
3. 컴퓨터 브라우저에서 http://localhost:8080 을 열면 "열린 어르신 방"에 태블릿 방이 보입니다. 눌러 조무사로 들어갑니다.
4. 태블릿에 "다리가 너무 아퍼"라고 말하면 안심 문장이 소리로 나오고, 조무사 브라우저 위에 빨간 위급 알림이 뜹니다.
5. 시계를 3초 누르면 보호자 화면으로 돌아옵니다.

에뮬레이터 마이크가 말을 못 알아들으면, 에뮬레이터 설정의 Microphone에서 "Virtual microphone uses host audio input"을 켭니다.

## 검사

```bash
flutter analyze && flutter test
# 실제 서버에 붙여 보기 (서버를 켜 둔 채로)
SILVERCARE_SERVER=http://localhost:8080 flutter test test/server_e2e_test.dart
```
