# 명세: 안전 (escalation · notify)

## 목적

어르신 말에서 낙상·통증·자해/타해 신호를 잡아 기록하고, 행동해야 할 사람(픽업 중이면 담당 조무사, 말동무 중이면 보호자)의 휴대폰에 바로 알립니다. 이 시스템에서 가장 먼저 지켜야 하는 부분입니다.

## 경계

- 한다: 키워드 규칙(1차 방어선), 위급 이벤트 저장(규칙·LLM), FCM 푸시, 알림 확인(ack) 기록.
- 하지 않는다: 어르신에게 할 말 고르기(→ [server-conversation.md](./server-conversation.md) CONV-03), Claude의 판단(`flag_concern`은 2차 방어선, 대화 엔진이 받아 넘김), 앱 화면(→ [apps.md](./apps.md)).

## 파일

| 파일 | 내용 |
| --- | --- |
| `server/internal/escalation/rules.go` | `Rules`(정규식, 규칙 ID), `Check`, `harmless`(아파트 같은 오탐 단어 지우기) |
| `server/internal/escalation/event.go` | `FromRule`, `FromLLM` (escalation_event 저장) |
| `server/internal/notify/escalation.go` | `EscalationWorker`(River, 큐 `alerts`), `EscalationMessage` |
| `server/internal/notify/fcm.go` | `Sender`, `FCM`(HTTP v1), `Unavailable`, 알림 채널 |
| `server/internal/httpapi/server.go`, `guardian.go` | 미확인 목록·상세·ack API |
| 테스트 | `escalation/rules_test.go`, `notify/notify_test.go`, `httpapi/stage5_test.go`, `stage6_test.go` |

## 인터페이스

| 방향 | 이름 | 설명 |
| --- | --- | --- |
| 제공 | `escalation.Check(text) (Match, bool)` | 규칙을 순서대로 보고 첫 번째 일치 |
| 제공 | `escalation.FromRule`, `FromLLM` | 이벤트 저장. `notified_targets`는 `[]`로 시작 |
| 제공 | `notify.EscalationWorker` | `jobs.EscalationAlertArgs` 처리 |
| 기대 | `notify.Sender.Send(ctx, token, Message)` | `ErrUnavailable`(설정 없음), `ErrUnregistered`(죽은 토큰) 구분 |
| 기대 | `session.Alerter.EscalationRaised` | `jobs.Enqueuer`가 작업을 넣음 |

## 요구사항

- `SAFE-01`: 모든 어르신 발화는 Claude보다 먼저 `Check`를 거칩니다. 규칙은 목록 순서대로 보고 첫 일치가 이깁니다.
- `SAFE-02`: 규칙에는 바꾸지 않는 ID가 있고 `escalation_event.rule_id`에 남습니다. 패턴을 바꾸면 새 ID를 씁니다.
- `SAFE-03`: 오탐 단어(`아파트`)는 매칭 전에 공백으로 지웁니다 (RE2에 부정 전방탐색이 없음).
- `SAFE-04`: 트리거 종류는 `FALL_MENTION`, `PAIN_COMPLAINT`, `SELF_OR_OTHER_HARM`, `OTHER_ANOMALY`(LLM만)입니다. LLM이 모르는 종류를 보내면 `OTHER_ANOMALY`로 저장합니다.
- `SAFE-05`: 이벤트마다 알림 작업이 하나입니다 (이벤트 ID로 유일, 최대 6번, 약 25분 재시도).
- `SAFE-06`: 받는 사람: 세션에 담당 조무사가 있으면(픽업) 그 조무사의 휴대폰 전부, 없으면(말동무) 보호자의 휴대폰 전부.
- `SAFE-07`: 한 대라도 받으면 성공이고 받은 기기를 `notified_targets`에 기록합니다. 아무도 못 받았으면 오류를 돌려 재시도합니다. 죽은 토큰의 기기는 해지합니다.
- `SAFE-08`: 이미 확인된(ack) 이벤트는 재시도 때 보내지 않습니다.
- `SAFE-09`: 푸시가 없어도 앱이 30초마다 지난 24시간의 미확인 이벤트를 읽어 보여줍니다 (`GET /escalations/open`, `GET /guardian/escalations/open`).
- `SAFE-10`: 위급 알림은 Android 채널 `escalation`(높은 중요도, 소리)과 높은 우선순위로 보냅니다. 하루 소식은 다른 채널입니다.
- `SAFE-11`: 위급 규칙이나 알림을 바꾸는 PR은 감지돼야 하는 문장과 감지되면 안 되는 문장을 테스트에 넣습니다.

## 설정값

| 환경변수 | 기본 | 뜻 |
| --- | --- | --- |
| `FCM_CREDENTIALS_FILE` | 없음 | Firebase 서비스 계정 JSON 경로. 없으면 푸시 없이 앱 목록만 |

규칙 목록은 코드(`Rules`)에 있습니다. 단계 7에서 요양·의료 전문가 검토로 확정합니다.

## 실패할 때

| 상황 | 동작 |
| --- | --- |
| 이벤트 저장 실패 | 어르신은 안심 문장을 듣고, 오류 로그만 (알림 없음) |
| 작업 등록 실패 | 오류 로그, 앱 목록에는 보임 ([CR-05](../code-review.md#cr-05)) |
| FCM 미설정 | 경고 로그, 성공 처리 (재시도 안 함) |
| 받을 휴대폰 없음 | 경고 로그, 성공 처리 ([CR-04](../code-review.md#cr-04)) |
| FCM 일시 오류 | 재시도 (River 백오프) |

## 확장 지점

- 규칙 추가: `Rules`에 새 ID로 한 줄 + 테스트 두 종류. 오탐 단어는 `harmless`에.
- 다른 푸시 수단(문자, 카카오 알림톡): `notify.Sender` 구현을 추가하고 `EscalationWorker`가 여러 Sender를 차례로 쓰게 합니다.
- 받는 사람 규칙 변경: `EscalationWorker.Work`의 분기(SAFE-06)만 바꾸면 됩니다.

## 알려진 문제

- [CR-01](../code-review.md#cr-01): 흔한 활용형을 놓침 (넘어져, 아퍼, 아팠어 등).
- [CR-02](../code-review.md#cr-02): Claude의 `flag_concern`이 잘린 답에서 버려짐.
- [CR-04](../code-review.md#cr-04): 조무사 휴대폰이 없으면 아무도 못 받음.
- [CR-05](../code-review.md#cr-05): 이벤트 저장과 작업 등록이 따로임.

## 다시 만들 때

- 규칙 필터를 LLM 판단으로 대체하지 않습니다 (architecture.md 안전장치 결정). LLM은 더하는 쪽만 합니다.
- 오탐보다 미탐이 위험합니다. 패턴은 넓게 잡고 오탐은 `harmless`와 테스트로 줄입니다.
- 알림을 다시 만들면 "누가 언제 받았는지"(`notified_targets`)와 "누가 확인했는지"(`acknowledged_by_*`)를 계속 남깁니다. 파일럿 사후 검토에 필요합니다.
