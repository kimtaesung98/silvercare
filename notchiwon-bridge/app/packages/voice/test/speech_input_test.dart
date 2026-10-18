import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:voice/voice.dart';

/// 테스트가 "들린 말"을 정해 주는 음성 인식 엔진.
class FakeRecognizer implements SpeechRecognizerPort {
  var listens = 0;
  var cancels = 0;
  Completer<String?>? _current;
  void Function()? _onSpeech;
  void Function(double level)? onLevel;

  /// 다음 listen에서 던질 오류 (한 번만).
  Object? failNext;

  bool get listening => _current != null;

  @override
  Future<String?> listen({
    void Function()? onSpeech,
    void Function(double level)? onLevel,
  }) {
    listens++;
    final error = failNext;
    if (error != null) {
      failNext = null;
      return Future.error(error);
    }
    _onSpeech = onSpeech;
    this.onLevel = onLevel;
    return (_current = Completer<String?>()).future;
  }

  @override
  Future<void> cancel() async {
    cancels++;
    _end(null);
  }

  /// 어르신이 [text]라고 말하고 멈춘 것으로 칩니다.
  Future<void> say(String text) async {
    _onSpeech?.call();
    _end(text);
    await pumpEventQueue();
  }

  /// 아무 말 없이 한 번 끝난 것으로 칩니다.
  Future<void> silence() async {
    _end(null);
    await pumpEventQueue();
  }

  void _end(String? text) {
    final current = _current;
    _current = null;
    if (current != null && !current.isCompleted) current.complete(text);
  }
}

void main() {
  late FakeRecognizer recognizer;
  late DeviceSpeechInput input;
  late List<String> heard;
  late int started;

  setUp(() async {
    recognizer = FakeRecognizer();
    input = DeviceSpeechInput(recognizer, retryDelay: Duration.zero);
    heard = [];
    started = 0;
    input.heard.listen((h) => heard.add((h as HeardText).text));
    input.speechStarted.listen((_) => started++);
    await input.start();
    await pumpEventQueue();
  });

  tearDown(() => input.close());

  test('알아들은 말을 글자로 내보내고 다시 듣는다', () async {
    await recognizer.say(' 오늘 병원 다녀왔어 ');

    expect(heard, ['오늘 병원 다녀왔어']);
    expect(started, 1);
    expect(recognizer.listens, 2);
    expect(recognizer.listening, isTrue);
  });

  test('아무 말이 없으면 내보내지 않고 계속 듣는다', () async {
    await recognizer.silence();
    await recognizer.say('   ');

    expect(heard, isEmpty);
    expect(recognizer.listens, 3);
  });

  test('AI가 말하는 동안에는 엔진을 쉬게 하고, 끝나면 다시 듣는다', () async {
    input.muted = true;
    await pumpEventQueue();

    expect(recognizer.cancels, 1);
    expect(recognizer.listening, isFalse);
    expect(recognizer.listens, 1);

    input.muted = false;
    await pumpEventQueue();

    expect(recognizer.listening, isTrue);
    expect(recognizer.listens, 2);
  });

  test('엔진이 실패하면 쉬었다가 다시 듣는다', () async {
    recognizer.failNext = StateError('음성 인식 없음');
    await recognizer.silence();

    expect(recognizer.listens, 3);
    expect(recognizer.listening, isTrue);
  });

  test('그만 들으면 엔진을 멈추고 더 듣지 않는다', () async {
    await input.stop();
    await pumpEventQueue();

    expect(input.isListening, isFalse);
    expect(recognizer.listening, isFalse);
    expect(recognizer.listens, 1);
  });

  test('엔진이 알려준 소리 크기를 levels로 내보낸다', () async {
    final levels = <double>[];
    final sub = input.levels.listen(levels.add);
    await input.start();
    await pumpEventQueue();

    recognizer.onLevel!(0.4);
    await pumpEventQueue();

    expect(levels, [0.4]);
    await sub.cancel();
  });

  test('안드로이드 소리 크기(dB)를 0~1로 바꾼다', () {
    expect(SpeechToTextRecognizer.soundLevel(-5), 0);
    expect(SpeechToTextRecognizer.soundLevel(4), 0.5);
    expect(SpeechToTextRecognizer.soundLevel(20), 1);
  });

  test('녹음기도 같은 듣기 장치로 쓸 수 있다', () async {
    final mic = _SilentMic();
    final recorder = UtteranceRecorder(mic);
    expect(recorder, isA<SpeechInput>());
    await recorder.close();
  });
}

class _SilentMic implements MicrophonePort {
  @override
  Future<Stream<Never>> start() async => const Stream.empty();

  @override
  Future<void> stop() async {}
}
