import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:api_client/api_client.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:stream_channel/stream_channel.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

/// 테스트용 채널: 서버가 보낸 프레임을 흉내 내고 태블릿이 보낸 것을 모읍니다.
class FakeChannel extends StreamChannelMixin<dynamic>
    implements WebSocketChannel {
  FakeChannel()
    : _incoming = StreamController<dynamic>(),
      _outgoing = StreamController<dynamic>.broadcast();

  final StreamController<dynamic> _incoming;
  final StreamController<dynamic> _outgoing;
  final List<Object?> sent = [];

  void serverSends(Object frame) => _incoming.add(frame);

  Future<void> serverCloses() => _incoming.close();

  @override
  Stream<dynamic> get stream => _incoming.stream;

  @override
  WebSocketSink get sink => _FakeSink(this);

  @override
  Future<void> get ready => Future.value();

  @override
  int? get closeCode => null;

  @override
  String? get closeReason => null;

  @override
  String? get protocol => null;

  @override
  void pipe(StreamChannel<dynamic> other) => throw UnimplementedError();
}

class _FakeSink implements WebSocketSink {
  _FakeSink(this.channel);

  final FakeChannel channel;

  @override
  void add(dynamic data) {
    channel.sent.add(data);
    channel._outgoing.add(data);
  }

  @override
  void addError(Object error, [StackTrace? stackTrace]) {}

  @override
  Future<void> addStream(Stream<dynamic> stream) async {}

  @override
  Future<void> close([int? closeCode, String? closeReason]) async {
    await channel._incoming.close();
  }

  @override
  Future<void> get done => channel._outgoing.done;
}

String frame(String type, Map<String, dynamic> data) =>
    jsonEncode({'type': type, 'data': data});

const sessionId = '9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d';

void main() {
  group('ElderEvent.decode', () {
    test('서버 → 태블릿 이벤트를 읽는다', () {
      final ready = ElderEvent.decode(
        frame('connection.ready', {
          'deviceId': 'd',
          'elderId': 'e',
          'activeSession': {'sessionId': sessionId, 'mode': 'COMPANION'},
          'openerVersion': 'v1',
        }),
      ) as ConnectionReady;
      expect(ready.openerVersion, 'v1');
      expect(ready.activeSession?.mode, SessionMode.COMPANION);

      final opener = ElderEvent.decode(
        frame('ai.opener', {
          'sessionId': sessionId,
          'turnId': 2,
          'clipId': 'c',
          'category': 'ESCALATION',
        }),
      ) as AiOpener;
      expect(opener.category, OpenerCategory.ESCALATION);

      final end = ElderEvent.decode(
        frame('ai.turn_end', {
          'sessionId': sessionId,
          'turnId': 2,
          'outcome': 'FALLBACK',
          'sentences': 1,
        }),
      ) as AiTurnEnd;
      expect(end.outcome, TurnOutcome.fallback);
    });

    test('모르는 이벤트와 모르는 enum 값은 그냥 넘어간다', () {
      expect(
        ElderEvent.decode(frame('something.new', {'a': 1})),
        isA<UnknownEvent>(),
      );
      final opener = ElderEvent.decode(
        frame('ai.opener', {
          'sessionId': sessionId,
          'turnId': 1,
          'clipId': 'c',
          'category': 'NEW_KIND',
        }),
      ) as AiOpener;
      expect(opener.category, OpenerCategory.RECALL);
    });

    test('봉투가 아니면 FormatException', () {
      expect(() => ElderEvent.decode('nope'), throwsFormatException);
      expect(() => ElderEvent.decode('{"type":"ping"}'), throwsFormatException);
    });
  });

  group('ElderSocket', () {
    late FakeChannel channel;
    late List<FakeChannel> opened;
    late ElderSocket socket;

    Future<void> connect() async {
      opened = [];
      socket = ElderSocket(
        serverUrl: Uri.parse('http://localhost:8000'),
        token: 't',
        connect: (_, _) async {
          channel = FakeChannel();
          opened.add(channel);
          return channel;
        },
        initialBackoff: const Duration(milliseconds: 10),
        maxBackoff: const Duration(milliseconds: 20),
      );
      await socket.start();
    }

    test('주소는 ws로 바뀌고 /ws/elder를 가리킨다', () async {
      await connect();
      expect(socket.socketUrl.toString(), 'ws://localhost:8000/ws/elder');
      expect(
        ElderSocket(
          serverUrl: Uri.parse('https://example.kr'),
          token: 't',
        ).socketUrl.toString(),
        'wss://example.kr/ws/elder',
      );
      await socket.close();
    });

    test('오디오가 붙은 ai.reply는 바이너리 프레임까지 모아서 내보낸다', () async {
      await connect();
      final events = <ElderEvent>[];
      socket.events.listen(events.add);

      channel.serverSends(
        frame('ai.reply', {
          'sessionId': sessionId,
          'turnId': 2,
          'index': 1,
          'utteranceId': 'u',
          'text': '안녕하세요.',
          'audio': {'format': 'mp3', 'bytes': 3},
        }),
      );
      await pumpEventQueue();
      expect(events, isEmpty, reason: '오디오를 기다리는 동안은 내보내지 않는다');

      channel.serverSends(Uint8List.fromList([1, 2, 3]));
      await pumpEventQueue();
      final reply = events.single as AiReply;
      expect(reply.text, '안녕하세요.');
      expect(reply.audio, [1, 2, 3]);
      await socket.close();
    });

    test('오디오가 null이면 바로 내보낸다', () async {
      await connect();
      final events = <ElderEvent>[];
      socket.events.listen(events.add);
      channel.serverSends(
        frame('ai.reply', {
          'sessionId': sessionId,
          'turnId': 2,
          'index': 1,
          'utteranceId': 'u',
          'text': '안녕하세요.',
          'audio': null,
        }),
      );
      await pumpEventQueue();
      expect((events.single as AiReply).audio, isNull);
      await socket.close();
    });

    test('ping에 자동으로 pong을 보낸다', () async {
      await connect();
      final events = <ElderEvent>[];
      socket.events.listen(events.add);
      channel.serverSends(frame('ping', {'nonce': 'n-1'}));
      await pumpEventQueue();
      expect(events, isEmpty);
      expect(jsonDecode(channel.sent.single as String), {
        'type': 'pong',
        'ts': anything,
        'data': {'nonce': 'n-1'},
      });
      await socket.close();
    });

    test('elder.audio는 텍스트 다음에 바이너리를 보낸다', () async {
      await connect();
      socket.send(
        ElderCommand.audio(
          sessionId: sessionId,
          clientId: 'u-1',
          audio: Uint8List.fromList([9, 9]),
        ),
        id: 'm-1',
      );
      expect(channel.sent, hasLength(2));
      final sent = jsonDecode(channel.sent.first as String) as Map;
      expect(sent['type'], 'elder.audio');
      expect(sent['id'], 'm-1');
      expect((sent['data'] as Map)['audio'], {
        'format': 'pcm16',
        'sampleRateHz': 16000,
        'bytes': 2,
      });
      expect(channel.sent.last, [9, 9]);
      await socket.close();
    });

    test('끊기면 다시 붙는다', () async {
      await connect();
      final states = <ConnectionState>[];
      socket.states.listen(states.add);
      final first = channel;
      await first.serverCloses();
      await Future<void>.delayed(const Duration(milliseconds: 50));
      expect(states, contains(ConnectionState.disconnected));
      expect(socket.state, ConnectionState.connected);
      expect(opened, hasLength(2), reason: '새 연결을 연다');
      expect(identical(channel, first), isFalse);

      // 새 연결로 이벤트가 계속 온다.
      final events = <ElderEvent>[];
      socket.events.listen(events.add);
      channel.serverSends(frame('session.rejected', {'reason': 'BEDTIME'}));
      await pumpEventQueue();
      expect(events.single, isA<SessionRejected>());
      await socket.close();
    });

    test('close 뒤에는 다시 붙지 않는다', () async {
      await connect();
      await socket.close();
      await channel.serverCloses();
      await Future<void>.delayed(const Duration(milliseconds: 50));
      expect(socket.state, ConnectionState.disconnected);
      expect(opened, hasLength(1));
    });
  });
}
