import 'dart:async';
import 'dart:math' as math;
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:voice/voice.dart';

/// 20ms(16kHz PCM16 = 640바이트)짜리 조각을 만듭니다. [level]은 0~1.
Uint8List chunk(double level, {int ms = 20}) {
  final samples = 16000 * ms ~/ 1000;
  final pcm = Int16List(samples);
  for (var i = 0; i < samples; i++) {
    // 사인파라 RMS는 진폭의 1/√2입니다.
    pcm[i] = (math.sin(i / 4) * level * math.sqrt2 * 32767).round().clamp(
      -32768,
      32767,
    );
  }
  return pcm.buffer.asUint8List();
}

class FakeMic implements MicrophonePort {
  final controller = StreamController<Uint8List>.broadcast();
  var starts = 0;
  var stops = 0;

  @override
  Future<Stream<Uint8List>> start() async {
    starts++;
    return controller.stream;
  }

  @override
  Future<void> stop() async => stops++;
}

void main() {
  group('SpeechSegmenter', () {
    test('조용하다 → 말한다 → 조용해지면 한 구간이 끝난다', () {
      final s = SpeechSegmenter();
      expect(s.add(chunk(0.01)), isNull);
      expect(s.isSpeaking, isFalse);

      expect(s.add(chunk(0.3)), SpeechEvent.started);
      for (var i = 0; i < 30; i++) {
        expect(s.add(chunk(0.3)), isNull, reason: '말하는 중');
      }
      // 900ms(45조각) 조용해야 끝납니다.
      SpeechEvent? event;
      for (var i = 0; i < 60 && event == null; i++) {
        event = s.add(chunk(0.001));
      }
      expect(event, SpeechEvent.ended);
      expect(s.isSpeaking, isFalse);
      expect(s.buffered, isNotEmpty);
    });

    test('짧은 소리는 버린다', () {
      final s = SpeechSegmenter();
      expect(s.add(chunk(0.5)), SpeechEvent.started);
      SpeechEvent? event;
      for (var i = 0; i < 60 && event == null; i++) {
        event = s.add(chunk(0.001));
      }
      expect(event, SpeechEvent.tooShort);
      expect(s.buffered, isEmpty);
    });

    test('말 중간에 잠깐 쉬는 것으로는 끊기지 않는다', () {
      final s = SpeechSegmenter();
      s.add(chunk(0.3));
      for (var i = 0; i < 25; i++) {
        s.add(chunk(0.3));
      }
      for (var i = 0; i < 20; i++) {
        expect(s.add(chunk(0.001)), isNull, reason: '400ms 쉼은 아직 발화 중');
      }
      expect(s.add(chunk(0.3)), isNull);
      expect(s.isSpeaking, isTrue);
    });

    test('너무 길면 거기서 끊는다', () {
      final s = SpeechSegmenter(
        config: const SegmenterConfig(
          maxUtterance: Duration(milliseconds: 600),
        ),
      );
      expect(s.add(chunk(0.3)), SpeechEvent.started);
      SpeechEvent? event;
      for (var i = 0; i < 40 && event == null; i++) {
        event = s.add(chunk(0.3));
      }
      expect(event, SpeechEvent.ended);
    });

    test('rms는 조용한 조각과 큰 조각을 가른다', () {
      expect(SpeechSegmenter.rms(chunk(0.001)), lessThan(0.05));
      expect(SpeechSegmenter.rms(chunk(0.5)), greaterThan(0.08));
    });
  });

  group('UtteranceRecorder', () {
    test('발화가 끝날 때마다 PCM을 내보낸다', () async {
      final mic = FakeMic();
      final recorder = UtteranceRecorder(mic);
      final utterances = <Uint8List>[];
      final starts = <void>[];
      recorder.utterances.listen(utterances.add);
      recorder.speechStarted.listen(starts.add);

      await recorder.start();
      expect(mic.starts, 1);
      for (var i = 0; i < 30; i++) {
        mic.controller.add(chunk(0.3));
      }
      for (var i = 0; i < 60; i++) {
        mic.controller.add(chunk(0.001));
      }
      await pumpEventQueue();
      expect(starts, hasLength(1));
      expect(utterances, hasLength(1));
      expect(utterances.single.length, greaterThan(640 * 25));

      await recorder.stop();
      expect(mic.stops, 1);
      await recorder.close();
    });

    test('AI가 말하는 동안에는 마이크 입력을 무시한다', () async {
      final mic = FakeMic();
      final recorder = UtteranceRecorder(mic);
      final utterances = <Uint8List>[];
      recorder.utterances.listen(utterances.add);
      await recorder.start();
      recorder.muted = true;
      for (var i = 0; i < 30; i++) {
        mic.controller.add(chunk(0.3));
      }
      for (var i = 0; i < 60; i++) {
        mic.controller.add(chunk(0.001));
      }
      await pumpEventQueue();
      expect(utterances, isEmpty);
      await recorder.close();
    });
  });
}
