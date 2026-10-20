# 실버케어 v2

방향과 구조는 [../docs/DIRECTION.md](../docs/DIRECTION.md)를 먼저 읽으세요. 이 폴더는 그 4절의 구조를 그대로 따릅니다.

| 층 | 폴더 | 하는 일 |
| --- | --- | --- |
| 바탕 | `chat/` | 대화방, 참가자, 메시지, 실시간 전달. 서비스 내용은 모름 |
| 참가자 | `companion/` | 말동무 AI. 사람과 같은 방식으로 방에 들어와 읽고 씀. 무엇을 말할지는 `Brain`이 정함 |
| 모듈 | `safety/` | 위급 필터. 사람 말을 AI보다 먼저 보고 표시(`Tags`)를 붙임. 안전 지킴이 참가자가 AI 대신 안심 문장으로 답하고, 화면이 방의 다른 사람에게 빨간 알림을 띄움 |
| 서비스 | `cmd/server/` | 방을 만든 어르신 곁에 말동무 AI와 안전 지킴이를 넣고, 말동무는 어르신에게만 답하는 규칙. 그리고 HTTP·브라우저 화면 |
| 앱 | `apps/elder_tablet/` | 어르신 태블릿 (Flutter). 보호자가 켜 두면 마이크 원 하나로 말만 주고받음. 서버 채팅에 어르신 참가자로 들어감. [README](apps/elder_tablet/README.md) |

## 해 보기

```bash
cd v2
go run ./cmd/server                 # http://localhost:8080
ANTHROPIC_API_KEY=... go run ./cmd/server   # Claude가 답함 (COMPANION_MODEL로 모델 변경)
```

1. 브라우저에서 http://localhost:8080 을 열고 "대화 시작"을 누릅니다. 어르신으로 방에 들어갑니다.
2. 말을 적으면 말동무 AI가 답합니다. 키가 없으면 정해진 맞장구로 답합니다.
3. 위쪽 "다른 사람 초대 링크"를 새 창에서 열면 조무사로 같은 방에 들어옵니다.
4. 어르신 창에 "다리가 너무 아퍼"처럼 적으면 말동무 대신 안전 지킴이가 안심 문장으로 답하고, 조무사 창 위에 빨간 위급 알림이 뜹니다.

태블릿 앱으로 해 보는 법은 [apps/elder_tablet/README.md](apps/elder_tablet/README.md)에 있습니다. 첫 화면 아래 "열린 어르신 방"에서 태블릿이 연 방에 조무사로 들어갈 수 있습니다.

새 모듈은 `chat.Module`(메시지에 표시 붙이기)과, 필요하면 표시를 보고 움직이는 참가자로 만듭니다. `safety/`가 그 예입니다.

## 검사

```bash
cd v2 && gofmt -l . && go vet ./... && go test -race ./...
```

지금은 메모리에만 저장하므로 서버를 끄면 대화가 사라집니다.
