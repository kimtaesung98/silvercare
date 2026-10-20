import 'dart:async';

import 'package:flutter_tts/flutter_tts.dart';
import 'package:speech_to_text/speech_recognition_error.dart';
import 'package:speech_to_text/speech_to_text.dart';

/// 태블릿의 귀와 입. 지금은 Android에 들어 있는 음성 인식과 읽어 주기를 쓰고,
/// 서버 음성 모듈이 생기면 이 자리에 다른 구현을 끼웁니다.
abstract class Voice {
  /// 마이크 권한을 묻고 음성 인식을 준비합니다. 쓸 수 있으면 true.
  Future<bool> init();

  /// 한 번 듣습니다. 말이 들리기 시작하면 [onHeard]를 부르고, 말이 끝나면 들은 글을,
  /// 아무 말도 없거나 실패하면 null을 돌려줍니다.
  Future<String?> listen({required void Function() onHeard});

  /// 듣던 것을 멈춥니다. 진행 중인 [listen]은 곧 끝납니다.
  Future<void> stopListening();

  /// 글을 소리로 읽고, 다 읽으면 끝납니다.
  Future<void> speak(String text);
}

/// Android 기본 음성 인식(speech_to_text)과 TTS(flutter_tts).
class DeviceVoice implements Voice {
  final _stt = SpeechToText();
  final _tts = FlutterTts();
  bool _ready = false;
  Completer<String?>? _current;
  String _last = '';

  @override
  Future<bool> init() async {
    if (_ready) return true;
    _ready = await _stt.initialize(onStatus: _onStatus, onError: _onError);
    await _tts.setLanguage('ko-KR');
    await _tts.setSpeechRate(0.45); // 어르신이 알아듣기 쉽게 조금 천천히
    await _tts.awaitSpeakCompletion(true);
    return _ready;
  }

  @override
  Future<String?> listen({required void Function() onHeard}) async {
    if (!_ready && !await init()) return null;
    final c = _current = Completer<String?>();
    _last = '';
    var heard = false;
    await _stt.listen(
      listenOptions: SpeechListenOptions(
        localeId: 'ko_KR',
        listenFor: const Duration(seconds: 30),
        pauseFor: const Duration(seconds: 3),
        partialResults: true,
        cancelOnError: true,
      ),
      onResult: (r) {
        _last = r.recognizedWords;
        if (!heard && _last.isNotEmpty) {
          heard = true;
          onHeard();
        }
        if (r.finalResult) _finish(_last);
      },
    );
    return c.future;
  }

  void _onStatus(String status) {
    if (status == SpeechToText.doneStatus || status == SpeechToText.notListeningStatus) {
      // 마지막 결과가 status보다 늦게 올 때가 있어 잠깐 기다립니다.
      Future.delayed(const Duration(milliseconds: 300), () => _finish(_last));
    }
  }

  void _onError(SpeechRecognitionError e) => _finish(null);

  void _finish(String? text) {
    final c = _current;
    if (c == null || c.isCompleted) return;
    _current = null;
    c.complete(text == null || text.trim().isEmpty ? null : text.trim());
  }

  @override
  Future<void> stopListening() async {
    await _stt.stop();
    _finish(_last);
  }

  @override
  Future<void> speak(String text) async {
    await _tts.speak(text);
  }
}
