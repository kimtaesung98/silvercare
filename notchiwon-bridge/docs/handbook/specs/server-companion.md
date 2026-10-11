# 명세: 말동무 모드 (companion)

## 목적

하원 뒤 어르신이 버튼을 누르거나 보호자가 정한 시각이 되면 태블릿 대화를 엽니다. 무응답·취침 시간·하루 토큰 한도로 끝내고, 하루가 끝나면 보호자에게 하루 소식을 보냅니다.

## 경계

- 한다: 시작 가능 여부(정책), 분 단위 tick(안부 대화 시작, 조용한 세션 종료, 하루 소식 큐), 하루 소식 작성·발송, 긴 대화 요약 작업.
- 하지 않는다: 대화 턴(→ [server-conversation.md](./server-conversation.md)), 위급 알림(→ [server-safety.md](./server-safety.md), 말동무 세션은 보호자에게 감), 설정 API 입력 검사(→ `httpapi`, [server-platform.md](./server-platform.md)).

## 파일

| 파일 | 내용 |
| --- | --- |
| `server/internal/companion/schedule.go` | `Schedule`(설정 + 시간대), `Policy`(`CanStart`, `Exhausted`, `TokensToday`), `Clock` |
| `server/internal/companion/tick.go` | `TickWorker`: `closeStale`, `startDue`, `queueDigests` |
| `server/internal/companion/digest.go` | `DigestWorker`: 하루 소식 작성(Haiku 또는 인용), 저장, 푸시 |
| `server/internal/companion/history.go` | `HistoryWorker`: 긴 대화 앞부분 요약 저장 |
| `server/internal/session/goodbye.go` | "그만하자" 류 문구 (대화 엔진에서 씀) |
| 테스트 | `companion/companion_test.go`, `tick_test.go`, `digest_test.go`, `history_test.go`, `session/companion_test.go`, `httpapi/stage6_test.go` |

## 인터페이스

| 방향 | 이름 | 설명 |
| --- | --- | --- |
| 제공 | `Policy.CanStart(ctx, elderID) (reason string, err)` | `""`이면 가능, 아니면 `COMPANION_DISABLED`·`BEDTIME`·`TOKEN_LIMIT` (`session.HubConfig.CanStart`) |
| 제공 | `Policy.Exhausted` | 대화 엔진의 `Budget` |
| 기대 | `Tablets` | `Connected`, `CompanionStarted`, `End` (`session.Hub`가 구현) |
| 기대 | `Queue.Digest(ctx, elderID, date)` | `jobs.Enqueuer` |
| 기대 | `llm.Digester`, `llm.Compacter`, `notify.Sender` | 키가 없으면 대체 동작 |

## 요구사항

- `COMP-01`: 말동무 세션은 설정이 꺼져 있거나, 취침 시간이거나, 오늘 토큰을 다 썼거나, 이미 열린 세션이 있으면 시작하지 않습니다 (`session.rejected`).
- `COMP-02`: 취침 시간은 자정을 넘을 수 있고(21:00~07:00), 시작과 끝이 같으면 취침 시간이 없습니다.
- `COMP-03`: 하루 토큰은 어르신 시간대의 자정부터 센 모든 세션(픽업 포함)의 입출력 토큰 합입니다. 한도는 설정값, 없으면 `COMPANION_DAILY_TOKEN_LIMIT`.
- `COMP-04`: tick은 1분마다 한 번(River 주기 작업, 리더 인스턴스만) 돌고 30초 안에 끝나야 합니다. 한 부분이 실패해도 나머지는 돕니다.
- `COMP-05`: 열린 말동무 세션은 취침 시간이 되면 `BEDTIME`, 마지막 활동에서 `COMPANION_IDLE_AFTER`가 지나면 `NO_RESPONSE`로 끝냅니다.
- `COMP-06`: 안부 시각이 지난 지 5분 안이고, 태블릿이 연결돼 있고, 그 시각 이후 시작한 안부 세션이 없고, COMP-01을 통과하면 `startedBy=SCHEDULE` 세션을 열고 태블릿에 알립니다. 연결이 없으면 열지 않습니다 (아무도 안 듣는 대화에 토큰을 쓰지 않음).
- `COMP-07`: 하루 소식은 취침 시작 시각(없으면 `COMPANION_DIGEST_AT`)에 (어르신, 날짜)당 하나만 만듭니다. 재시도해도 다시 쓰거나 다시 알리지 않습니다.
- `COMP-08`: 대화가 없던 날은 "대화가 없었어요"를, Claude가 없으면 어르신 말 마지막 몇 개를 인용합니다. 그날 위급이 있었으면 "위급 감지: … (바로 연락드렸어요)"를 붙입니다.
- `COMP-09`: 긴 대화 요약은 앞으로만 나아갑니다 (`history_summary_through`가 줄지 않음). Claude가 없으면 요약 없이 그대로 보냅니다.

## 설정값

| 환경변수 | 기본 | 뜻 |
| --- | --- | --- |
| `COMPANION_DAILY_TOKEN_LIMIT` | `60000` | 기본 하루 한도 |
| `COMPANION_IDLE_AFTER` | `5m` | 무응답 종료 |
| `COMPANION_DIGEST_AT` | `21:00` | 취침 시간이 없을 때 하루 소식 시각 |

어르신별 설정은 `companion_schedule` 행(켜기, 안부 시각 목록, 취침 시작·끝, 시간대, 하루 한도)이고 보호자 앱이나 조무사 API로 고칩니다.

## 실패할 때

| 상황 | 동작 |
| --- | --- |
| 설정 읽기 실패 | 그 어르신만 건너뛰고 다음 분에 다시 |
| 토큰 합 읽기 실패 (턴 중) | 대화를 계속함 (끊는 것보다 나음), 오류 로그 |
| 하루 소식 푸시 실패 | 재시도. 보호자 휴대폰이 없으면 앱에서 읽음 |

## 확장 지점

- 새 종료 조건: `TickWorker.closeStale`에 분기 + `ended_reason` CHECK 제약에 값 추가(새 마이그레이션) + 태블릿 `_rejectionText`/종료 문구.
- 시작 정책 변경: `Policy.canStart`만 바꾸면 버튼과 안부 대화에 함께 적용됩니다.

## 알려진 문제

- [CR-03](../code-review.md#cr-03): 긴 대화 요약이 세션당 한 번만 들어감 (COMP-09 실질 위반).
- [CR-08](../code-review.md#cr-08): 설정 행이 없는 어르신은 하루 소식·안부 대화가 없음.
- [CR-09](../code-review.md#cr-09): 취침 시작이 자정 이후면 날짜가 밀림.
- [CR-12](../code-review.md#cr-12): 설정을 고치는 길이 둘.

## 다시 만들 때

- 시간 계산은 항상 어르신 설정의 시간대(`Schedule.Location`)로 합니다. 서버 시간대를 쓰면 하루 경계가 틀어집니다.
- tick이 늦게 돌아도 지난 일을 한꺼번에 하지 않도록 5분 창(`jobs.TickEvery*5`)을 둡니다. 창을 없애면 서버 재시작 때 지난 안부 대화가 몰려서 시작됩니다.
