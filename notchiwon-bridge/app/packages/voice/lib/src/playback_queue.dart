import 'dart:async';
import 'dart:typed_data';

/// 재생할 소리 한 조각.
class PlayItem {
  const PlayItem({
    required this.turnId,
    required this.index,
    this.clipId,
    this.audio,
    this.text,
  });

  /// 캐시된 0번 문장·추임새 (음성 파일은 [clipId]로 찾습니다).
  const PlayItem.clip({
    required int turnId,
    required int index,
    required String clipId,
    String? text,
  }) : this(turnId: turnId, index: index, clipId: clipId, text: text);

  /// 서버가 보낸 MP3.
  const PlayItem.bytes({
    required int turnId,
    required int index,
    required Uint8List audio,
    String? text,
  }) : this(turnId: turnId, index: index, audio: audio, text: text);

  /// 이 조각이 속한 AI 턴.
  final int turnId;

  /// 턴 안의 문장 번호. 0번은 0번 문장, 추임새는 -1입니다.
  final int index;

  final String? clipId;
  final Uint8List? audio;

  /// 화면에 띄울 글자 (음성이 없을 때도 보여줍니다).
  final String? text;
}

/// 소리 하나를 끝까지 재생하는 장치. 실제 구현은 [JustAudioPlayer]이고,
/// 테스트는 가짜를 씁니다.
abstract class AudioPlayerPort {
  /// [item]을 재생하고 끝날 때까지 기다립니다. 중간에 [stop]이 불리면
  /// 일찍 돌아옵니다.
  Future<void> play(PlayItem item);

  /// 재생 중인 소리를 멈춥니다.
  Future<void> stop();
}

/// 0번 → 1번 → 2번 순서로 트는 재생 큐 (architecture.md 6.1절 7번).
///
/// 서버는 문장을 만드는 대로 보내므로 도착 순서와 재생 순서가 다를 수 있습니다.
/// 큐는 받은 순서대로 쌓아 한 번에 하나씩 틀고, 어르신이 말을 시작하면
/// ([cancelTurn]) 그 턴의 남은 문장을 버립니다.
class PlaybackQueue {
  PlaybackQueue(this._player);

  final AudioPlayerPort _player;
  final _queue = <PlayItem>[];
  final _started = StreamController<PlayItem>.broadcast();
  final _finished = StreamController<PlayItem>.broadcast();

  PlayItem? _current;
  int? _cancelledTurn;
  bool _running = false;
  bool _closed = false;

  /// 재생을 시작한 조각 (화면 글자를 바꿀 때 씁니다).
  Stream<PlayItem> get started => _started.stream;

  /// 재생이 끝난 조각.
  Stream<PlayItem> get finished => _finished.stream;

  /// 재생 중인 조각.
  PlayItem? get current => _current;

  /// 아직 틀지 않은 조각 수.
  int get pending => _queue.length;

  /// 재생 중이거나 기다리는 조각이 있는지.
  bool get isBusy => _running || _queue.isNotEmpty;

  /// 큐에 넣습니다. 이미 취소된 턴의 조각은 버립니다.
  void add(PlayItem item) {
    if (_closed || item.turnId == _cancelledTurn) return;
    _queue.add(item);
    unawaited(_drain());
  }

  /// 그 턴의 남은 조각을 버리고, 그 턴을 재생 중이면 멈춥니다.
  /// 끼어들기(`elder.barge_in`)와 세션 종료에 씁니다.
  Future<void> cancelTurn(int turnId) async {
    _cancelledTurn = turnId;
    _queue.removeWhere((i) => i.turnId == turnId);
    if (_current?.turnId == turnId) {
      await _player.stop();
    }
  }

  /// 전부 버리고 멈춥니다.
  Future<void> clear() async {
    _queue.clear();
    _cancelledTurn = _current?.turnId;
    await _player.stop();
  }

  Future<void> close() async {
    _closed = true;
    await clear();
    await _started.close();
    await _finished.close();
  }

  Future<void> _drain() async {
    if (_running) return;
    _running = true;
    try {
      while (_queue.isNotEmpty) {
        final item = _queue.removeAt(0);
        if (item.turnId == _cancelledTurn) continue;
        _current = item;
        if (!_started.isClosed) _started.add(item);
        try {
          await _player.play(item);
        } on Object {
          // 한 문장을 못 틀어도 대화는 이어집니다.
        }
        _current = null;
        if (!_finished.isClosed) _finished.add(item);
      }
    } finally {
      _running = false;
    }
  }
}
