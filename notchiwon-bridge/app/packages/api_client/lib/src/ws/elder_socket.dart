import 'dart:async';
import 'dart:typed_data';

import 'package:web_socket_channel/web_socket_channel.dart';

import 'events.dart';

/// 서버 연결 상태.
enum ConnectionState { disconnected, connecting, connected }

/// 연결 하나를 여는 함수. 테스트는 가짜 채널을 돌려줍니다.
typedef ChannelFactory = Future<WebSocketChannel> Function(
  Uri url,
  String token,
);

/// `/ws/elder` 클라이언트.
///
/// - 텍스트 프레임을 [ElderEvent]로 읽고, `audio`가 있는 이벤트는 뒤따라오는
///   바이너리 프레임을 붙여 한 번에 내보냅니다.
/// - `ping`에는 자동으로 `pong`을 보냅니다.
/// - 끊기면 1초부터 30초까지 늘려가며 다시 붙습니다. 태블릿은 벽에 붙어 하루
///   종일 켜져 있으므로 재연결을 포기하지 않습니다.
class ElderSocket {
  ElderSocket({
    required this.serverUrl,
    required this.token,
    ChannelFactory? connect,
    this.initialBackoff = const Duration(seconds: 1),
    this.maxBackoff = const Duration(seconds: 30),
  }) : _connect = connect ?? _defaultConnect;

  /// 서버 주소 (`http://`나 `https://`). `/ws/elder`는 알아서 붙습니다.
  final Uri serverUrl;

  /// 기기 토큰.
  final String token;

  final ChannelFactory _connect;
  final Duration initialBackoff;
  final Duration maxBackoff;

  final _events = StreamController<ElderEvent>.broadcast();
  final _states = StreamController<ConnectionState>.broadcast();

  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _sub;
  Timer? _retry;
  Duration _backoff = Duration.zero;
  bool _closed = false;
  ElderEvent? _awaitingAudio;
  var _state = ConnectionState.disconnected;

  /// 서버가 보낸 이벤트.
  Stream<ElderEvent> get events => _events.stream;

  /// 연결 상태 변화.
  Stream<ConnectionState> get states => _states.stream;

  ConnectionState get state => _state;

  static Future<WebSocketChannel> _defaultConnect(Uri url, String token) async {
    final channel = WebSocketChannel.connect(url, protocols: null);
    await channel.ready;
    return channel;
  }

  /// WebSocket 주소 (`ws://host/ws/elder`). 토큰은 헤더로 보낼 수 없는 환경을
  /// 대비해 쿼리로도 붙이지 않습니다. [ChannelFactory]가 헤더를 넣습니다.
  Uri get socketUrl => serverUrl.replace(
    scheme: serverUrl.scheme == 'https' ? 'wss' : 'ws',
    path: '/ws/elder',
  );

  /// 연결을 시작합니다. 이미 붙어 있으면 아무것도 하지 않습니다.
  Future<void> start() async {
    if (_closed || _channel != null || _state == ConnectionState.connecting) {
      return;
    }
    _setState(ConnectionState.connecting);
    try {
      final channel = await _connect(socketUrl, token);
      if (_closed) {
        await channel.sink.close();
        return;
      }
      _channel = channel;
      _backoff = Duration.zero;
      _setState(ConnectionState.connected);
      _sub = channel.stream.listen(
        _onFrame,
        onError: (Object _) => _reconnect(),
        onDone: _reconnect,
        cancelOnError: true,
      );
    } on Object {
      _reconnect();
    }
  }

  /// 메시지를 보냅니다. 연결이 없으면 버립니다 (끊긴 동안의 발화는 의미가 없습니다).
  void send(ElderCommand command, {String? id}) {
    final channel = _channel;
    if (channel == null) return;
    channel.sink.add(command.encode(id: id));
    final audio = command.audio;
    if (audio != null) {
      channel.sink.add(audio);
    }
  }

  /// 연결을 닫고 다시 붙지 않습니다.
  Future<void> close() async {
    _closed = true;
    _retry?.cancel();
    await _sub?.cancel();
    await _channel?.sink.close();
    _channel = null;
    _setState(ConnectionState.disconnected);
    await _events.close();
    await _states.close();
  }

  void _onFrame(dynamic frame) {
    if (frame is List<int>) {
      final pending = _awaitingAudio;
      _awaitingAudio = null;
      if (pending != null) {
        _emit(pending.withAudio(Uint8List.fromList(frame)));
      }
      return;
    }
    if (frame is! String) return;

    // 오디오를 기다리다 텍스트가 오면 오디오 없이 내보냅니다.
    final pending = _awaitingAudio;
    _awaitingAudio = null;
    if (pending != null) _emit(pending);

    final ElderEvent event;
    try {
      event = ElderEvent.decode(frame);
    } on FormatException {
      return;
    }
    if (event is Ping) {
      send(ElderCommand.pong(event.nonce));
      return;
    }
    if (event.expectedAudioBytes != null) {
      _awaitingAudio = event;
      return;
    }
    _emit(event);
  }

  void _emit(ElderEvent event) {
    if (!_events.isClosed) _events.add(event);
  }

  void _setState(ConnectionState state) {
    _state = state;
    if (!_states.isClosed) _states.add(state);
  }

  void _reconnect() {
    if (_closed) return;
    _sub?.cancel();
    _sub = null;
    _channel = null;
    _awaitingAudio = null;
    _setState(ConnectionState.disconnected);
    _backoff = _backoff == Duration.zero
        ? initialBackoff
        : Duration(
            milliseconds: (_backoff.inMilliseconds * 2).clamp(
              1,
              maxBackoff.inMilliseconds,
            ),
          );
    _retry?.cancel();
    _retry = Timer(_backoff, start);
  }
}
