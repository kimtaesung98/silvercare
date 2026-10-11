# 노치원 브릿지 핸드북

> 기준일: 2026-10-11 (master `8110f59`, PR #13까지 반영).
> 사람은 한눈에 구조를 보고, 에이전트는 고치거나 다시 만들 때 출발점으로 쓰는 문서 묶음입니다.

## 무엇을 먼저 읽나

| 하고 싶은 일 | 읽을 문서 |
| --- | --- |
| 지금 어디까지 왔는지 | [status.md](./status.md) |
| 시스템 속을 그림으로 보기 | [whitebox.md](./whitebox.md) |
| 한 모듈을 고치거나 다시 만들기 | [specs/](./specs/) 중 해당 파일 |
| 알려진 문제와 고칠 순서 | [code-review.md](./code-review.md) |
| 왜 이렇게 만들었는지 | [../architecture.md](../architecture.md) 7절 결정 표 |
| 대화 한 턴의 규칙 원문 | [../specs/conversation-loop.md](../specs/conversation-loop.md) |
| WebSocket 계약 원문 | [../../server/api/ws-events.md](../../server/api/ws-events.md) |

## 명세서 목록

| 파일 | 다루는 코드 |
| --- | --- |
| [specs/contracts.md](./specs/contracts.md) | REST(`openapi.yaml`), WebSocket, DB 마이그레이션, 코드 생성 |
| [specs/server-conversation.md](./specs/server-conversation.md) | `internal/session`, `opener`, `llm`, `speech` |
| [specs/server-safety.md](./specs/server-safety.md) | `internal/escalation`, `notify` |
| [specs/server-pickup.md](./specs/server-pickup.md) | `internal/visit`, `eta`, `briefing` |
| [specs/server-companion.md](./specs/server-companion.md) | `internal/companion` |
| [specs/server-platform.md](./specs/server-platform.md) | `cmd/*`, `config`, `auth`, `httpapi`, `jobs`, `migrate`, `db` |
| [specs/apps.md](./specs/apps.md) | `app/apps/*`, `app/packages/*` |

## 명세서 형식 (에이전트용 규칙)

모든 명세서는 같은 순서로 씁니다. 새 모듈을 추가할 때도 이 틀을 복사합니다.

1. **목적**: 한두 문장.
2. **경계**: 하는 일과 하지 않는 일. 하지 않는 일은 누가 하는지 적습니다.
3. **파일**: 진입점과 테스트 위치.
4. **인터페이스**: 다른 모듈이 부르는 함수·타입, 이 모듈이 기대는 인터페이스.
5. **요구사항**: `ID: 규칙` 형식. ID는 `영역-번호`(예: `CONV-03`)이고 지우거나 다시 쓰지 않습니다. 규칙을 바꾸면 새 ID를 붙이고 옛 것에 `(폐기, CONV-12로 대체)`를 적습니다.
6. **설정값**: 환경변수와 기본값.
7. **실패할 때**: 외부 서비스나 DB가 실패하면 어떻게 되는지.
8. **확장 지점**: 바꿔 끼울 수 있는 인터페이스와 바꾸는 방법.
9. **알려진 문제**: [code-review.md](./code-review.md)의 `CR-xx` 링크.
10. **다시 만들 때**: 지켜야 할 불변식과 함정.

## 에이전트가 이 문서로 일할 때

- 코드를 바꾸기 전에 해당 명세의 요구사항 ID를 확인하고, PR 설명에 바뀌는 ID를 적습니다.
- 요구사항을 바꾸는 변경이면 명세도 같은 PR에서 고칩니다. 설계를 바꾸면 [architecture.md](../architecture.md) 7절에 결정을 추가합니다.
- 리뷰 항목을 고치면 [code-review.md](./code-review.md)에서 그 항목의 상태를 `고침 (PR #n)`으로 바꿉니다.
- 위급 필터·알림에 닿는 변경은 감지돼야 하는 문장과 감지되면 안 되는 문장을 테스트에 함께 넣습니다 ([development-process.md](../development-process.md) 3절).
- 확인 명령: 서버 `cd server && make test`(Postgres 필요, `TEST_DATABASE_URL`), 앱 `cd app && flutter analyze && flutter test`(패키지별), 계약 `cd server && make generate` 후 `git diff`가 비어야 합니다.
