# 실버케어 v2

방향과 구조는 [../docs/DIRECTION.md](../docs/DIRECTION.md)를 먼저 읽으세요. 이 폴더는 그 4절의 구조를 그대로 따릅니다.

| 층 | 폴더 | 하는 일 |
| --- | --- | --- |
| 바탕 | `chat/` | 대화방, 참가자, 메시지, 실시간 전달. 서비스 내용은 모름 |
| 참가자 | `companion/` | 말동무 AI. 사람과 같은 방식으로 방에 들어와 읽고 씀. 무엇을 말할지는 `Brain`이 정함 |
| 모듈 | (2회차부터) | 위급 필터, 음성 등 |
| 서비스 | `cmd/server/` | 방을 만들면 말동무 AI를 함께 넣는 규칙 하나, 그리고 HTTP·브라우저 화면 |

## 해 보기

```bash
cd v2
go run ./cmd/server                 # http://localhost:8080
ANTHROPIC_API_KEY=... go run ./cmd/server   # Claude가 답함 (COMPANION_MODEL로 모델 변경)
```

1. 브라우저에서 http://localhost:8080 을 열고 "대화 시작"을 누릅니다. 어르신으로 방에 들어갑니다.
2. 말을 적으면 말동무 AI가 답합니다. 키가 없으면 정해진 맞장구로 답합니다.
3. 위쪽 "다른 사람 초대 링크"를 새 창에서 열면 조무사로 같은 방에 들어옵니다. 세 참가자가 한 방에서 대화합니다.

## 검사

```bash
cd v2 && gofmt -l . && go vet ./... && go test -race ./...
```

지금은 메모리에만 저장하므로 서버를 끄면 대화가 사라집니다.
