import 'playback_queue.dart';

/// 글자를 소리 내 읽는 장치. 실제 구현은 기기 내장 TTS([FlutterTtsSpeaker])이고,
/// 테스트는 가짜를 씁니다.
abstract class TextToSpeechPort {
  /// [text]를 읽고 다 읽을 때까지 기다립니다. 중간에 [stop]이 불리면 일찍
  /// 돌아옵니다.
  Future<void> speak(String text);

  Future<void> stop();
}

/// 음성 파일이 있으면 그것을 틀고, 없으면 글자를 기기 TTS로 읽는 재생기.
///
/// 서버에 Amazon Polly 설정이 없으면 0번 문장 음성도, 문장마다 오는 MP3도 없습니다.
/// 그래도 어르신이 글자만 보지 않고 소리로 듣도록 기기 목소리로 대신 읽습니다.
class SpeakingPlayer implements AudioPlayerPort {
  SpeakingPlayer({
    required this.audio,
    required this.hasAudio,
    required this.tts,
  });

  /// 음성 파일을 트는 재생기.
  final AudioPlayerPort audio;

  /// [audio]로 틀 음성이 있는지.
  final bool Function(PlayItem item) hasAudio;

  /// 음성이 없을 때 글자를 읽는 TTS.
  final TextToSpeechPort tts;

  @override
  Future<void> play(PlayItem item) async {
    if (hasAudio(item)) return audio.play(item);
    final text = item.text?.trim() ?? '';
    // 글자 없는 추임새는 소리 없이 넘어갑니다.
    if (text.isEmpty) return;
    await tts.speak(text);
  }

  @override
  Future<void> stop() async {
    await Future.wait([audio.stop(), tts.stop()]);
  }
}
