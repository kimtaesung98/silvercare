# 명세: 앱 (elder_tablet · caregiver · guardian · packages)

## 목적

서버 기능을 세 사람에게 보여 줍니다. 어르신은 태블릿에 말만 하고, 조무사는 이동 중 위치를 보내고 브리핑·위급을 받고, 보호자는 하루 소식·위급·말동무 설정을 다룹니다. 앱에는 규칙을 두지 않고, 판단은 서버가 합니다.

## 경계

- 한다: 화면, 녹음·인식·재생, 서버 연결과 재접속, 로그인 토큰 보관, 위치 수집, 푸시 수신.
- 하지 않는다: 위급 판단, 세션 시작 조건, ETA 계산, 요약 (모두 서버). API 키를 들고 있지 않음 (CONV-17).

## 파일

Flutter 워크스페이스는 `app/pubspec.yaml` 하나로 묶여 있습니다. 상태 관리는 Riverpod(조무사·보호자)와 `ChangeNotifier`(태블릿), 화면 이동은 go_router입니다.

| 경로 | 내용 |
| --- | --- |
| `app/apps/elder_tablet/lib/main.dart` | dart-define(`SERVER_URL`, `DEVICE_TOKEN`, `SPEECH`)으로 부품 조립 |
| `app/apps/elder_tablet/lib/src/session_controller.dart` | 소켓 이벤트 → 화면 상태(`idle`·`talking`·`arriving`), 재생 큐, 입력 켜고 끄기, 끼어들기, 지표 |
| `app/apps/elder_tablet/lib/src/kiosk.dart` + `android/.../MainActivity.kt`, `SessionService.kt` | Lock Task 화면 고정, Foreground Service |
| `app/apps/elder_tablet/lib/src/screens/` | 대기(말동무 버튼), 대화(자막), 도착 임박, 소리 막대(`mic_meter.dart`) |
| `app/apps/caregiver/lib/src/trip.dart` | `TripController`(위치 전송 반복), `GeolocatorSource`(위치 Foreground Service) |
| `app/apps/caregiver/lib/src/screens/` | 로그인, 오늘 방문, 방문(지도·도착 버튼), 브리핑 카드, 위급 |
| `app/apps/caregiver/lib/src/push.dart`, `credential_store.dart`, `backend.dart` | FCM, 보안 저장소 토큰 보관, API 묶음 |
| `app/apps/guardian/lib/src/screens/` | 로그인, 홈, 어르신(하루 소식), 위급, 말동무 설정 |
| `app/packages/api_client/` | 생성된 REST 클라이언트(`lib/src/generated`, 손대지 않음) + `ws/elder_socket.dart`, `ws/events.dart`(손으로 관리) |
| `app/packages/voice/` | `SpeechInput`(`DeviceSpeechInput`, `UtteranceRecorder`), `SpeechSegmenter`, `PlaybackQueue`, `SpeakingPlayer`, `ClipCache`, 플랫폼 어댑터(`platform.dart`) |
| `app/packages/ui/` | 어르신용 큰 글씨 테마, 큰 버튼, 자막판, 알림 줄 |
| 테스트 | 각 패키지 `test/` (가짜 부품 `fakes.dart`로 위젯·컨트롤러 테스트) |

## 인터페이스

| 방향 | 이름 | 설명 |
| --- | --- | --- |
| 제공 | `ElderSocket` | `events`(서버 이벤트 스트림), `states`(연결 상태), `send(ElderCommand)`, 1초→30초 지수 백오프 재접속 |
| 제공 | `SpeechInput` | `heard`(`HeardText` 또는 `HeardAudio`), `speechStarted`, `muted`, `start`/`stop` |
| 제공 | `AudioPlayerPort` / `PlaybackQueue` | 턴·문장 순서대로 재생, `cancelTurn` |
| 기대 | 서버 REST·WS | [contracts.md](./contracts.md) |
| 기대 | 플랫폼 | 마이크(`record`), 기기 음성 인식(`speech_to_text`), 기기 TTS(`flutter_tts`), 재생(`just_audio`), 위치(`geolocator`), FCM |

모든 플랫폼 기능은 `*Port` 인터페이스 뒤에 있어서 테스트는 가짜로 돌고, 실제 기기 어댑터는 `platform.dart`, `trip.dart`, `push.dart`에만 있습니다.

## 요구사항

### 태블릿

- `APP-01`: 태블릿은 기기 토큰 하나로 REST와 `/ws/elder`에 붙습니다. 끊기면 1초부터 두 배씩 최대 30초 간격으로 다시 붙고, `connection.ready.activeSession`이 있으면 그 세션을 이어갑니다.
- `APP-02`: 화면은 서버 이벤트로만 바뀝니다. 세션이 없으면 `idle`, 있으면 `talking`, 픽업 세션에서 ETA가 `arrivingMinutes`(2분) 이하면 `arriving`입니다.
- `APP-03`: 0번 문장은 `ClipCache`에 미리 받아 둔 클립을 틀고, 목록 버전(`openerVersion`)이 바뀌면 다시 받습니다. 클립이 없으면 건너뜁니다.
- `APP-04`: `ai.reply`에 MP3가 있으면 틀고, 없으면 기기 TTS로 읽습니다 (`SpeakingPlayer`). 재생 순서는 턴 번호·문장 번호 순입니다.
- `APP-05`: `SPEECH=device`(기본)면 기기 음성 인식 결과를 `elder.text`로, `SPEECH=server`면 말 구간을 잘라(무음 0.9초, 최소 0.4초, 최대 30초) `elder.audio`로 보냅니다.
- `APP-06`: AI가 말하는 동안은 기본으로 입력을 끕니다 (`muteWhileSpeaking`, 스피커 소리를 어르신 말로 알아듣지 않게). 그래서 끼어들기(`elder.barge_in`)는 이 값을 끈 경우에만 일어납니다.
- `APP-07`: `session.rejected` 사유는 한국어 문장으로 보여 줍니다 (`_rejectionText`). 새 사유를 서버에 더하면 여기도 더합니다.
- `APP-08`: 태블릿은 키오스크로 고정하고, 화면이 꺼져도 대화가 이어지게 Foreground Service를 켭니다.
- `APP-09`: 턴마다 지연 지표(발화 끝 → 0번, 0번 끝 → 1번)를 `client.metrics`로 보냅니다.
- `APP-17`: 마이크는 앱이 켜질 때 열고 세션이 끝나도 닫지 않습니다. 대기·대화 화면의 소리 막대(`MicMeter`)가 소리 크기를 보여 주고, 마이크를 못 열면 "마이크를 쓸 수 없어요"를 보여 줍니다. 세션이 없을 때 말이 들리면 말동무를 요청하고 그 말을 들고 있다가 세션이 열리면 보냅니다. 거절되면 버리고 `requestCooldown`(30초) 동안 다시 요청하지 않습니다 (결정 43).

### 조무사 앱

- `APP-10`: 출발하면 10초마다 위치를 `POST /visits/{id}/location`으로 보냅니다. 위치는 Foreground Service로 받아 화면이 꺼져도 이어집니다. 도착 버튼은 `POST /visits/{id}/arrive`.
- `APP-11`: 도착 뒤 브리핑이 아직 없으면 3초마다 최대 40번 다시 묻습니다.
- `APP-12`: 홈은 30초마다 방문과 미확인 위급을 다시 읽습니다 (SAFE-09). 위급 푸시를 누르면 위급 화면으로 가고, 확인하면 ack를 보냅니다.
- `APP-13`: 로그인 토큰은 Android Keystore로 암호화해 보관하고, 401이면 로그인 화면으로 돌아갑니다.

### 보호자 앱

- `APP-14`: 홈은 30초마다 미확인 위급을 다시 읽고, 어르신 화면에서 하루 소식 목록을 봅니다.
- `APP-15`: 말동무 설정(켜기, 안부 시각, 취침 시간, 하루 한도)을 `PUT /guardian/elders/{id}/companion-schedule`로 고칩니다.
- `APP-16`: FCM 토큰은 로그인 뒤 `PUT /guardian/devices/me/fcm-token`으로 등록합니다 (조무사는 `/devices/me/fcm-token`).

## 설정값

| 값 | 기본 | 위치 | 뜻 |
| --- | --- | --- | --- |
| `SERVER_URL` | `http://10.0.2.2:8000` | dart-define (세 앱) | 서버 주소 (에뮬레이터 기준) |
| `DEVICE_TOKEN` | 없음 | dart-define (태블릿) | `cmd/admin create-tablet`이 준 토큰 |
| `SPEECH` | `device` | dart-define (태블릿) | `device` 또는 `server` |
| `arrivingMinutes` | 2 | `SessionController` | 도착 임박 화면 |
| `muteWhileSpeaking` | true | `SessionController` | APP-06 |
| `startOnSpeech` | true | `SessionController` | APP-17 |
| `requestCooldown` | 30초 | `SessionController` | APP-17 |
| 위치 간격 | 10초 | `reportIntervalProvider` | APP-10 |
| 브리핑 재시도 | 3초 × 40 | `briefingPollProvider` | APP-11 |
| 목록 새로고침 | 30초 | 각 `home_screen.dart` | APP-12, APP-14 |

실행 예시는 `scripts/dev.sh`와 루트 `.env.example`에 있습니다.

## 실패할 때

| 상황 | 동작 |
| --- | --- |
| 서버 연결 끊김 | 알림 줄에 연결 상태, 백오프 재접속 (APP-01) |
| 마이크·인식 권한 없음 | 앱을 켤 때 묻습니다 (`main.dart`, 설치 때 한 번 허용해 둠) |
| 기기 음성 인식 오류 | 2초 뒤 다시 듣기 (`DeviceSpeechInput.retryDelay`) |
| 서버 음성 없음 | 기기 TTS (APP-04) |
| `STT_FAILED` | "잘 못 들었어요. 한 번만 더 말씀해 주세요." |
| 푸시 미설정 | `NoPush`, 30초 새로고침으로 대신 |
| 위치 권한 거부 | 위치를 보내지 않음, 서버는 예정 시각 ETA |

## 확장 지점

- 새 WS 이벤트: `events.dart`의 `decode`/`ElderCommand` + `SessionController._onEvent` (`UnknownEvent`는 무시되므로 순서 상관없이 배포 가능).
- 다른 인식 엔진(예: 온디바이스 Whisper): `SpeechRecognizerPort` 또는 `SpeechInput`을 구현하고 `main.dart`에서 고릅니다.
- 다른 재생기: `AudioPlayerPort`. 서버 MP3와 클립 파일을 틀 수 있어야 합니다.
- 화면 문구: 어르신 화면은 `packages/ui` 테마(큰 글씨)를 씁니다. 새 화면도 같은 위젯을 씁니다.

## 알려진 문제

- [CR-06](../code-review.md#cr-06): `playedIndex`를 보내지만 서버가 쓰지 않음.
- [CR-11](../code-review.md#cr-11): `DEVICE_TOKEN`이 APK에 들어감.
- [CR-14](../code-review.md#cr-14): 기기 음성 모드를 실제 태블릿에서 아직 검증하지 않음.
- 주석 정리 필요: `main.dart`의 `DEVICE_TOKEN` 주석은 없는 경로(`POST /devices/{deviceId}/token`)를, `session_controller.dart`의 `TabletInfo` 주석은 `GET /tablet/context`를 가리킵니다. 실제는 `cmd/admin create-tablet`과 `GET /tablet/me`입니다.

## 다시 만들 때

- 앱을 다시 짜도 부품 경계(`SpeechInput` → `ElderSocket` → `PlaybackQueue`)는 유지하는 것이 좋습니다. 테스트가 이 경계에서 가짜를 끼웁니다.
- 생성 코드(`api_client/lib/src/generated`)는 고치지 않고 `make generate`로만 바꿉니다 (CTR-01).
- 어르신 화면은 글자를 줄이지 않고, 버튼은 하나에 가깝게 둡니다.
