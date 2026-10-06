import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:voice/voice.dart';

/// 테스트용 재생기: 틀기 시작한 조각을 기록하고, 테스트가 끝내 줄 때까지 기다립니다.
class FakePlayer implements AudioPlayerPort {
  final played = <String>[];
  Completer<void>? _playing;

  @override
  Future<void> play(PlayItem item) async {
    played.add('${item.turnId}.${item.index}');
    _playing = Completer<void>();
    await _playing!.future;
  }

  @override
  Future<void> stop() async {
    if (_playing != null && !_playing!.isCompleted) _playing!.complete();
  }

  /// 재생 중인 소리가 끝난 것으로 칩니다.
  Future<void> finish() async {
    await stop();
    await pumpEventQueue();
  }
}

PlayItem clip(int turn, int index) =>
    PlayItem.clip(turnId: turn, index: index, clipId: 'c$index');

PlayItem reply(int turn, int index) => PlayItem.bytes(
  turnId: turn,
  index: index,
  audio: Uint8List.fromList([index]),
  text: '문장 $index',
);

void main() {
  test('0번 → 1번 → 2번 순서로 하나씩 튼다', () async {
    final player = FakePlayer();
    final queue = PlaybackQueue(player);
    final started = <String>[];
    queue.started.listen((i) => started.add('${i.turnId}.${i.index}'));

    queue
      ..add(clip(2, 0))
      ..add(reply(2, 1))
      ..add(reply(2, 2));
    await pumpEventQueue();
    expect(player.played, ['2.0'], reason: '한 번에 하나만 튼다');
    expect(queue.pending, 2);

    await player.finish();
    expect(player.played, ['2.0', '2.1']);
    await player.finish();
    expect(player.played, ['2.0', '2.1', '2.2']);
    await player.finish();
    expect(queue.isBusy, isFalse);
    expect(started, ['2.0', '2.1', '2.2']);
  });

  test('재생 중에 도착한 문장도 순서대로 이어 튼다', () async {
    final player = FakePlayer();
    final queue = PlaybackQueue(player);
    queue.add(clip(2, 0));
    await pumpEventQueue();
    queue.add(reply(2, 1));
    await player.finish();
    expect(player.played, ['2.0', '2.1']);
  });

  test('끼어들면 그 턴의 남은 문장을 버리고 멈춘다', () async {
    final player = FakePlayer();
    final queue = PlaybackQueue(player);
    queue
      ..add(clip(2, 0))
      ..add(reply(2, 1))
      ..add(reply(2, 2));
    await pumpEventQueue();

    await queue.cancelTurn(2);
    await pumpEventQueue();
    expect(player.played, ['2.0'], reason: '1번, 2번은 틀지 않는다');
    expect(queue.isBusy, isFalse);

    // 끼어들기 뒤에 도착한 같은 턴의 문장도 버린다.
    queue.add(reply(2, 3));
    await pumpEventQueue();
    expect(player.played, ['2.0']);

    // 다음 턴은 정상적으로 튼다.
    queue.add(clip(4, 0));
    await pumpEventQueue();
    expect(player.played, ['2.0', '4.0']);
  });

  test('한 문장을 못 틀어도 다음 문장으로 넘어간다', () async {
    final queue = PlaybackQueue(BrokenPlayer());
    final finished = <int>[];
    queue.finished.listen((i) => finished.add(i.index));
    queue
      ..add(reply(1, 1))
      ..add(reply(1, 2));
    await pumpEventQueue();
    expect(finished, [1, 2]);
  });

  test('clear는 전부 버린다', () async {
    final player = FakePlayer();
    final queue = PlaybackQueue(player);
    queue
      ..add(clip(2, 0))
      ..add(reply(2, 1));
    await pumpEventQueue();
    await queue.clear();
    await pumpEventQueue();
    expect(player.played, ['2.0']);
    expect(queue.pending, 0);
  });
}

/// 늘 실패하는 재생기.
class BrokenPlayer implements AudioPlayerPort {
  @override
  Future<void> play(PlayItem item) async => throw StateError('재생 실패');

  @override
  Future<void> stop() async {}
}
