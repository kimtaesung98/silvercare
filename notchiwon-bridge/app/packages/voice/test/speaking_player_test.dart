import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:voice/voice.dart';

class RecordingPlayer implements AudioPlayerPort {
  final played = <PlayItem>[];
  var stops = 0;

  @override
  Future<void> play(PlayItem item) async => played.add(item);

  @override
  Future<void> stop() async => stops++;
}

class RecordingTts implements TextToSpeechPort {
  final spoken = <String>[];
  var stops = 0;

  @override
  Future<void> speak(String text) async => spoken.add(text);

  @override
  Future<void> stop() async => stops++;
}

void main() {
  late RecordingPlayer audio;
  late RecordingTts tts;
  late SpeakingPlayer player;

  // 받아 둔 0번 문장 음성은 clip-1뿐입니다.
  bool hasAudio(PlayItem item) => item.audio != null || item.clipId == 'clip-1';

  setUp(() {
    audio = RecordingPlayer();
    tts = RecordingTts();
    player = SpeakingPlayer(audio: audio, hasAudio: hasAudio, tts: tts);
  });

  test('음성이 있으면 음성을 튼다', () async {
    await player.play(
      PlayItem.bytes(
        turnId: 1,
        index: 1,
        audio: Uint8List.fromList([1]),
        text: '그러셨어요',
      ),
    );
    await player.play(
      const PlayItem.clip(turnId: 1, index: 0, clipId: 'clip-1', text: '아이고'),
    );

    expect(audio.played, hasLength(2));
    expect(tts.spoken, isEmpty);
  });

  test('음성이 없으면 글자를 기기 목소리로 읽는다', () async {
    await player.play(
      const PlayItem.clip(turnId: 1, index: 0, clipId: 'clip-2', text: '아이고'),
    );
    await player.play(const PlayItem(turnId: 1, index: 1, text: '병원은 어떠셨어요?'));

    expect(tts.spoken, ['아이고', '병원은 어떠셨어요?']);
    expect(audio.played, isEmpty);
  });

  test('음성도 글자도 없는 추임새는 조용히 넘어간다', () async {
    await player.play(
      const PlayItem.clip(turnId: 1, index: -1, clipId: 'filler-1'),
    );

    expect(tts.spoken, isEmpty);
    expect(audio.played, isEmpty);
  });

  test('멈추면 음성과 기기 목소리를 둘 다 멈춘다', () async {
    await player.stop();

    expect(audio.stops, 1);
    expect(tts.stops, 1);
  });
}
