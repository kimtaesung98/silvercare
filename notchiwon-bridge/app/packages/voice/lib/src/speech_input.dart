import 'dart:async';

import 'package:flutter/foundation.dart';

/// 어르신 말 한 마디. 서버가 알아들을 녹음이거나, 기기가 이미 알아들은 글자입니다.
sealed class Heard {
  const Heard();
}

/// 녹음한 발화 (16kHz 모노 PCM16). 서버가 음성 인식을 합니다 (`elder.audio`).
class HeardAudio extends Heard {
  const HeardAudio(this.pcm);

  final Uint8List pcm;
}

/// 기기가 알아들은 글자. 서버에는 글자로 보냅니다 (`elder.text`).
class HeardText extends Heard {
  const HeardText(this.text);

  final String text;
}

/// 어르신 말을 듣는 장치. 서버 음성 인식용 녹음기([UtteranceRecorder])와
/// 기기 음성 인식([DeviceSpeechInput]) 두 가지가 있습니다.
abstract class SpeechInput {
  /// 끝난 발화 한 마디.
  Stream<Heard> get heard;

  /// 어르신이 말을 시작했다는 신호 (끼어들기 판단에 씁니다).
  Stream<void> get speechStarted;

  /// AI가 말하는 동안 듣지 않을지.
  bool get muted;
  set muted(bool value);

  /// 듣기 시작합니다.
  Future<void> start();

  /// 그만 듣습니다.
  Future<void> stop();

  Future<void> close();
}

/// 기기 음성 인식 엔진 하나. 실제 구현은 `speech_to_text` 패키지를 쓰고
/// ([SpeechToTextRecognizer]), 테스트는 가짜를 씁니다.
abstract class SpeechRecognizerPort {
  /// 한 마디를 듣고 알아들은 글자를 돌려줍니다. 아무 말도 없었으면 null입니다.
  /// 말이 들리기 시작하면 [onSpeech]를 부릅니다. 엔진을 쓸 수 없으면 던집니다.
  Future<String?> listen({void Function()? onSpeech});

  /// 듣던 것을 버리고 [listen]을 곧바로 끝냅니다.
  Future<void> cancel();
}

/// 기기의 음성 인식으로 듣는 [SpeechInput]. 서버 음성 설정이 없어도 대화할 수
/// 있도록, 알아들은 글자를 서버에 보냅니다.
///
/// 기기 엔진은 한 번에 한 마디만 들으므로, 한 마디가 끝나면 다시 듣기를
/// 되풀이합니다. 듣지 않는 동안([muted]) 엔진을 쉬게 해서 태블릿이 내는
/// 소리를 어르신 말로 알아듣지 않습니다.
class DeviceSpeechInput implements SpeechInput {
  DeviceSpeechInput(
    this._recognizer, {
    this.retryDelay = const Duration(seconds: 2),
  });

  final SpeechRecognizerPort _recognizer;

  /// 엔진이 실패하면 이만큼 쉬고 다시 듣습니다.
  final Duration retryDelay;

  final _heard = StreamController<Heard>.broadcast();
  final _speechStarted = StreamController<void>.broadcast();

  bool _running = false;
  bool _muted = false;
  Completer<void>? _wake;
  Future<void>? _loop;

  @override
  Stream<Heard> get heard => _heard.stream;

  @override
  Stream<void> get speechStarted => _speechStarted.stream;

  @override
  bool get muted => _muted;

  @override
  set muted(bool value) {
    if (_muted == value) return;
    _muted = value;
    if (value) {
      unawaited(_recognizer.cancel());
    } else {
      _wakeUp();
    }
  }

  bool get isListening => _running;

  @override
  Future<void> start() async {
    if (_running) return;
    _running = true;
    _loop = _run();
  }

  @override
  Future<void> stop() async {
    if (!_running) return;
    _running = false;
    _wakeUp();
    await _recognizer.cancel();
    await _loop;
    _loop = null;
  }

  @override
  Future<void> close() async {
    await stop();
    await _heard.close();
    await _speechStarted.close();
  }

  void _wakeUp() {
    _wake?.complete();
    _wake = null;
  }

  Future<void> _run() async {
    while (_running) {
      if (_muted) {
        _wake = Completer<void>();
        await _wake!.future;
        continue;
      }
      String? text;
      try {
        text = await _recognizer.listen(onSpeech: _onSpeech);
      } on Object catch (e) {
        debugPrint('기기 음성 인식 실패: $e');
        await Future<void>.delayed(retryDelay);
        continue;
      }
      // 듣는 중에 AI가 말을 시작했거나 대화가 끝났으면 버립니다.
      if (!_running || _muted) continue;
      final said = text?.trim() ?? '';
      if (said.isNotEmpty && !_heard.isClosed) _heard.add(HeardText(said));
    }
  }

  void _onSpeech() {
    if (_running && !_muted && !_speechStarted.isClosed) {
      _speechStarted.add(null);
    }
  }
}
