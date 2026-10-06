/// 노치원 브릿지 서버 클라이언트.
///
/// REST 클라이언트와 모델은 `server/api/openapi.yaml`에서 생성합니다
/// (`server/`에서 `make generate`). 생성 코드는 직접 고치지 않습니다.
library;

export 'src/generated/api.dart';
export 'src/ws/elder_socket.dart';
export 'src/ws/events.dart';
