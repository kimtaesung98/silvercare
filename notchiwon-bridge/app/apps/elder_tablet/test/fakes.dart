import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:stream_channel/stream_channel.dart';
import 'package:voice/voice.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

/// 서버 대신 프레임을 보내고, 태블릿이 보낸 것을 모으는 채널.
class FakeChannel extends StreamChannelMixin<dynamic>
    implements WebSocketChannel {
  FakeChannel()
    : _incoming = StreamController<dynamic>(),
      _outgoing = StreamController<dynamic>.broadcast();

  final StreamController<dynamic> _incoming;
  final StreamController<dynamic> _outgoing;
  final List<Object?> sent = [];

  /// 태블릿이 보낸 텍스트 프레임을 봉투에서 꺼내 돌려줍니다.
  List<Map<String, dynamic>> get sentEvents => [
    for (final frame in sent)
      if (frame is String) jsonDecode(frame) as Map<String, dynamic>,
  ];

  /// 그 type으로 보낸 것 중 마지막 (없으면 null).
  Map<String, dynamic>? lastSent(String type) {
    final matching = sentEvents.where((e) => e['type'] == type);
    return matching.isEmpty ? null : matching.last;
  }

  void serverSends(Object frame) => _incoming.add(frame);

  /// 봉투를 씌워 보냅니다.
  void serverSendsEvent(String type, Map<String, dynamic> data) =>
      serverSends(jsonEncode({'type': type, 'data': data}));

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
    if (!channel._incoming.isClosed) await channel._incoming.close();
  }

  @override
  Future<void> get done => channel._outgoing.done;
}

/// 소리를 내지 않고 재생한 것만 적어 두는 재생기.
class FakePlayer implements AudioPlayerPort {
  final played = <PlayItem>[];
  Completer<void>? _playing;

  /// 재생이 끝나지 않고 기다릴지 (끼어들기 시험에 씁니다).
  var hold = false;

  @override
  Future<void> play(PlayItem item) async {
    played.add(item);
    if (!hold) return;
    _playing = Completer<void>();
    await _playing!.future;
  }

  /// 기다리던 재생을 끝냅니다.
  void finish() {
    _playing?.complete();
    _playing = null;
  }

  @override
  Future<void> stop() async => finish();
}

/// 테스트가 PCM 조각을 직접 밀어넣는 마이크.
class FakeMicrophone implements MicrophonePort {
  final _chunks = StreamController<Uint8List>.broadcast();
  var started = 0;
  var stopped = 0;

  @override
  Future<Stream<Uint8List>> start() async {
    started++;
    return _chunks.stream;
  }

  @override
  Future<void> stop() async => stopped++;

  void push(Uint8List chunk) => _chunks.add(chunk);

  /// 같은 세기의 PCM16 조각 (16kHz에서 [ms]밀리초).
  static Uint8List tone(double level, {int ms = 100}) {
    final samples = Int16List(16 * ms);
    final value = (level * 32767).round();
    for (var i = 0; i < samples.length; i++) {
      samples[i] = i.isEven ? value : -value;
    }
    return samples.buffer.asUint8List();
  }
}

/// 메모리에 두는 0번 문장 캐시.
class MemoryClipStore implements ClipStore {
  final files = <String, Uint8List>{};
  String? version;

  @override
  Future<Uint8List?> read(String clipId) async => files[clipId];

  @override
  Future<void> write(String clipId, Uint8List bytes) async =>
      files[clipId] = bytes;

  @override
  Future<Set<String>> ids() async => files.keys.toSet();

  @override
  Future<void> delete(String clipId) async => files.remove(clipId);

  @override
  Future<String?> readVersion() async => version;

  @override
  Future<void> writeVersion(String v) async => version = v;
}

/// 0번 문장 목록을 고정으로 돌려주는 통로.
class FakeClipSource implements ClipSource {
  FakeClipSource({this.version = 'v1', this.texts = const {}});

  String version;

  /// clipId별 글자.
  Map<String, String> texts;

  @override
  Future<ClipManifest> manifest() async => ClipManifest(
    version: version,
    voice: 'vdain',
    clips: [
      for (final entry in texts.entries)
        ClipInfo(
          id: entry.key,
          category: 'RECALL',
          text: entry.value,
          durationMs: 800,
        ),
    ],
  );

  @override
  Future<Uint8List?> audio(String clipId) async =>
      texts.containsKey(clipId) ? Uint8List.fromList([1, 2, 3]) : null;
}
