import 'package:flutter_test/flutter_test.dart';
import 'package:just_audio/just_audio.dart';
import 'package:voice/voice.dart';

void main() {
  test('끝까지 틀면 재생이 끝난다', () async {
    await JustAudioPlayer.untilDone(
      Stream.fromIterable([ProcessingState.ready, ProcessingState.completed]),
    );
  });

  test('중간에 멈춰도(idle) 재생이 끝난다', () async {
    // 예전에는 completed만 기다려서 여기서 영영 멈췄습니다.
    await JustAudioPlayer.untilDone(
      Stream.fromIterable([ProcessingState.ready, ProcessingState.idle]),
    ).timeout(const Duration(seconds: 1));
  });
}
