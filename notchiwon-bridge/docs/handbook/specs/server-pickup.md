# 명세: 픽업 브릿지 (visit · eta · briefing)

## 목적

조무사가 어르신 댁으로 가는 동안 위치로 ETA를 계산하고, 도착 15분 전부터 태블릿 대화(픽업 대기 세션)를 엽니다. 도착하면 대화를 끝내고 조무사가 읽을 브리핑을 만듭니다.

## 경계

- 한다: 오늘 방문 목록, 위치 저장과 ETA, 세션 시작(한 번만), 도착 처리, 브리핑 작성, 관심사 키워드 갱신.
- 하지 않는다: 대화 내용(→ [server-conversation.md](./server-conversation.md)), 위급 알림(→ [server-safety.md](./server-safety.md)), 방문 만들기·취소(아직 API 없음, `cmd/admin seed-demo`나 DB 작업).

## 파일

| 파일 | 내용 |
| --- | --- |
| `server/internal/visit/visit.go` | `Service`: `ListToday`, `Get`, `RecordLocation`, `Arrive`, `SessionNotifier`, `Notifiers` |
| `server/internal/eta/eta.go` | `Estimator`, `Schedule`(예정 시각까지 남은 분) |
| `server/internal/eta/kakao.go` | `Kakao`(카카오모빌리티 길찾기, 실패하면 `Fallback`) |
| `server/internal/briefing/briefing.go` | `Worker`(River, 큐 `briefings`): 요약, 감정 태그, 키워드 |
| `server/internal/llm/briefing.go` | Haiku 브리핑 요청과 JSON 스키마 |
| `server/db/queries/visits.sql`, `keywords.sql`, `briefings.sql` | 상태 전이 쿼리, 키워드 감쇠 |
| 테스트 | `visit/visit_test.go`(동시 20건 포함), `eta/*_test.go`, `briefing/briefing_test.go`, `httpapi/stage5_test.go` |

## 인터페이스

| 방향 | 이름 | 설명 |
| --- | --- | --- |
| 제공 | `visit.Service.RecordLocation(ctx, caregiverID, visitID, Location) (LocationResult, error)` | `ErrNotFound`(남의 방문), `ErrClosed`(끝난 방문) |
| 제공 | `visit.Service.Arrive` | 방문 완료 + 세션 `CAREGIVER_ARRIVED` |
| 기대 | `visit.SessionNotifier` | `SessionStarted`, `SessionEnded`, `EtaUpdated`. `session.Hub`(태블릿)와 `jobs.Enqueuer`(브리핑 큐)가 구현 |
| 기대 | `eta.Estimator.Minutes(ctx, Request) (int, error)` | 정수 분, 음수 없음 |
| 기대 | `llm.Summarizer` | 브리핑 요약 (키가 없으면 `Unavailable`) |

## 요구사항

- `PICK-01`: 위치를 받을 때마다 `caregiver_location`에 저장하고 방문의 `eta_current`를 갱신합니다. `SCHEDULED`면 `EN_ROUTE`로 바꿉니다.
- `PICK-02`: ETA가 `SESSION_TRIGGER_ETA_MINUTES` 이하이고 세션이 없으면, 한 트랜잭션에서 방문을 `SESSION_ACTIVE`로 차지(조건부 UPDATE)하고, 열린 말동무 세션을 `PREEMPTED`로 끝내고, 픽업 세션을 만듭니다. 동시에 몇 건이 와도 세션은 하나입니다.
- `PICK-03`: 태블릿과 큐에는 트랜잭션이 커밋된 뒤에 알립니다. 순서는 끝난 말동무 → 새 세션 → ETA입니다.
- `PICK-04`: 세션이 열린 뒤의 위치는 `caregiver.eta`로 태블릿에 갑니다. 분이 바뀔 때만 보냅니다 (`Hub.lastEta`).
- `PICK-05`: ETA는 어르신 댁 좌표가 있고 카카오 키가 있으면 실제 도로 기준, 아니면 예정 시각까지 남은 분입니다. 카카오 호출이 실패해도 위치 전송은 실패하지 않습니다.
- `PICK-06`: 도착하면 방문을 `COMPLETED`로, 열린 세션을 `CAREGIVER_ARRIVED`로 한 트랜잭션에서 바꾸고, 커밋 뒤 태블릿에 `session.ended`, 큐에 브리핑 작업을 넣습니다.
- `PICK-07`: 브리핑은 세션당 하나입니다. 재시도해도 키워드를 두 번 올리지 않습니다 (이미 있으면 그대로 둠).
- `PICK-08`: Claude가 없으면 요약은 고정 문구 + 어르신 말 인용이고, 위급이 있었으면 감정 플래그가 `UNUSUAL`입니다.
- `PICK-09`: 키워드 점수는 `score = score × 0.9 + 이번 언급 수`로 갱신합니다 (현재는 언급된 키워드만, [CR-15](../code-review.md#cr-15)).
- `PICK-10`: 조무사는 자기 방문과 그 세션의 브리핑·위급만 봅니다 (`caregiver_id` 확인).

## 설정값

| 환경변수 | 기본 | 뜻 |
| --- | --- | --- |
| `SESSION_TRIGGER_ETA_MINUTES` | `15` | 세션을 여는 ETA |
| `KAKAO_MOBILITY_API_KEY` | 없음 | 없으면 `eta.Schedule` |
| `TIME_ZONE` | `Asia/Seoul` | "오늘" 방문 목록의 기준 |
| 태블릿 도착 임박 화면 | 2분 | 앱 쪽 값 (`SessionController.arrivingMinutes`) |

## 실패할 때

| 상황 | 동작 |
| --- | --- |
| 카카오 실패 | 예정 시각 기준 ETA, 경고 로그 |
| 태블릿 오프라인 | 세션은 열리고, 태블릿이 연결되면 `connection.ready.activeSession`으로 이어받고 인사 |
| 브리핑 작업 실패 | River 재시도 5번 |
| 도착을 두 번 누름 | 두 번째는 `ErrClosed` |

## 확장 지점

- 다른 지도(티맵): `eta.Estimator` 구현을 추가하고 `main.go`에서 고릅니다.
- 방문 취소·변경 API: `visit.Service`에 메서드를 추가하고 `CANCELLED` 전이 쿼리를 씁니다. 열린 세션이 있으면 끝낼 사유(`ended_reason`)를 새로 정해야 합니다 (CHECK 제약 변경 = 새 마이그레이션).

## 알려진 문제

- [CR-15](../code-review.md#cr-15): 키워드 감쇠가 시간 기준이 아님.
- 방문 취소 API 없음 (요구사항 아님, 파일럿 운영 때 필요할 수 있음).

## 다시 만들 때

- "세션 하나"는 코드 잠금이 아니라 DB가 보장합니다: 조건부 UPDATE(`ClaimVisitSession`)와 부분 유니크 인덱스. 여러 서버 인스턴스에서도 성립합니다.
- 알림을 트랜잭션 안에서 보내지 않습니다. 롤백된 세션을 태블릿에 알리게 됩니다.
