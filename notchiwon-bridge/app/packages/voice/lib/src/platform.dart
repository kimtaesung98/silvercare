import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_tts/flutter_tts.dart';
import 'package:just_audio/just_audio.dart';
import 'package:path_provider/path_provider.dart';
import 'package:record/record.dart';
import 'package:speech_to_text/speech_recognition_error.dart';
import 'package:speech_to_text/speech_recognition_result.dart';
import 'package:speech_to_text/speech_to_text.dart';

import 'clip_cache.dart';
import 'playback_queue.dart';
import 'recorder.dart';
import 'speaking_player.dart';
import 'speech_input.dart';

/// 기기에서 실제로 소리를 내는 [AudioPlayerPort].
///
/// 0번 문장은 캐시 폴더의 파일을, 서버가 보낸 문장은 메모리의 MP3를 틉니다.
class JustAudioPlayer implements AudioPlayerPort {
  JustAudioPlayer({required this.store, AudioPlayer? player})
    : _player = player ?? AudioPlayer();

  /// 0번 문장 음성이 있는 곳.
  final FileClipStore store;
  var _replyFiles = 0;
  final AudioPlayer _player;

  /// 틀 음성이 있는지 (0번 문장은 캐시에 받아 두었는지).
  bool hasAudio(PlayItem item) {
    final clipId = item.clipId;
    if (clipId != null) return store.fileOf(clipId).existsSync();
    return item.audio != null;
  }

  @override
  Future<void> play(PlayItem item) async {
    final clipId = item.clipId;
    final bytes = item.audio;
    if (clipId != null) {
      final file = store.fileOf(clipId);
      if (!file.existsSync()) return; // 아직 못 받은 음성: 글자만 보여줍니다.
      await _player.setFilePath(file.path);
    } else if (bytes != null) {
      // just_audio는 파일이나 URL을 틉니다. 문장 하나는 수십 KB라 임시 파일로
      // 쓰는 편이 메모리 소스(실험 API)보다 단순합니다.
      final file = File(
        '${Directory.systemTemp.path}/reply_${_replyFiles++ % 4}.mp3',
      );
      await file.writeAsBytes(bytes, flush: true);
      await _player.setFilePath(file.path);
    } else {
      return;
    }
    await _player.play();
    await untilDone(_player.processingStateStream);
    await _player.stop();
  }

  /// 재생이 끝나거나 [stop]으로 멈출 때까지 기다립니다.
  ///
  /// 끝까지 튼 경우만 기다리면, 세션이 끝나거나 어르신이 끼어들어 [stop]이
  /// 불렸을 때(상태가 idle로 감) 영영 돌아오지 않습니다. 그러면 재생 큐가
  /// 멈춘 채로 남아 다음 대화의 AI 목소리가 나오지 않고, 마이크도 "AI가
  /// 말하는 중"으로 묶여 어르신 말을 보내지 않습니다.
  static Future<void> untilDone(Stream<ProcessingState> states) =>
      states.firstWhere(
        (s) => s == ProcessingState.completed || s == ProcessingState.idle,
        orElse: () => ProcessingState.idle,
      );

  @override
  Future<void> stop() => _player.stop();

  Future<void> dispose() => _player.dispose();
}

/// 0번 문장 음성을 앱 지원 폴더에 두는 [ClipStore].
class FileClipStore implements ClipStore {
  FileClipStore(this._dir);

  /// 기기의 앱 지원 폴더 아래 `opener_clips/`를 씁니다.
  static Future<FileClipStore> open() async {
    final base = await getApplicationSupportDirectory();
    final dir = Directory('${base.path}/opener_clips');
    await dir.create(recursive: true);
    return FileClipStore(dir);
  }

  final Directory _dir;

  File fileOf(String clipId) => File('${_dir.path}/$clipId.mp3');

  File get _versionFile => File('${_dir.path}/version.txt');

  @override
  Future<Uint8List?> read(String clipId) async {
    final file = fileOf(clipId);
    return file.existsSync() ? file.readAsBytes() : null;
  }

  @override
  Future<void> write(String clipId, Uint8List bytes) =>
      fileOf(clipId).writeAsBytes(bytes, flush: true);

  @override
  Future<Set<String>> ids() async => {
    for (final f in _dir.listSync())
      if (f is File && f.path.endsWith('.mp3'))
        f.uri.pathSegments.last.replaceAll('.mp3', ''),
  };

  @override
  Future<void> delete(String clipId) async {
    final file = fileOf(clipId);
    if (file.existsSync()) await file.delete();
  }

  @override
  Future<String?> readVersion() async =>
      _versionFile.existsSync() ? _versionFile.readAsString() : null;

  @override
  Future<void> writeVersion(String version) =>
      _versionFile.writeAsString(version, flush: true);
}

/// `record` 패키지로 16kHz 모노 PCM16을 내보내는 마이크.
class DeviceMicrophone implements MicrophonePort {
  DeviceMicrophone({AudioRecorder? recorder})
    : _recorder = recorder ?? AudioRecorder();

  final AudioRecorder _recorder;

  /// 마이크 권한이 있는지 (없으면 물어봅니다).
  Future<bool> hasPermission() => _recorder.hasPermission();

  @override
  Future<Stream<Uint8List>> start() => _recorder.startStream(
    const RecordConfig(
      encoder: AudioEncoder.pcm16bits,
      sampleRate: 16000,
      numChannels: 1,
      // 스피커 소리가 마이크로 되돌아오는 것을 기기가 줄여 줍니다.
      echoCancel: true,
      noiseSuppress: true,
      // AI 목소리를 틀면 재생기가 오디오 포커스를 가져갑니다. 기본값(pause)은
      // 그때 녹음을 멈추고 다시 켜지 않아서 첫 대답 뒤로 마이크가 죽습니다.
      // 마이크는 앱이 켜진 내내 들어야 하므로 포커스와 상관없이 계속 녹음합니다.
      audioInterruption: AudioInterruptionMode.none,
    ),
  );

  @override
  Future<void> stop() async {
    if (await _recorder.isRecording()) await _recorder.stop();
  }

  Future<void> dispose() => _recorder.dispose();
}

/// 기기 내장 TTS로 읽는 [TextToSpeechPort] (안드로이드는 Google TTS).
class FlutterTtsSpeaker implements TextToSpeechPort {
  FlutterTtsSpeaker({
    FlutterTts? tts,
    this.language = 'ko-KR',
    this.rate = 0.45,
  }) : _tts = tts ?? FlutterTts();

  final FlutterTts _tts;

  /// 읽을 말.
  final String language;

  /// 읽는 빠르기. 안드로이드는 0.5가 보통 빠르기이고, 어르신은 조금 느린
  /// 목소리를 더 잘 따라옵니다.
  final double rate;

  Future<void>? _ready;

  Future<void> _setUp() async {
    await _tts.setLanguage(language);
    await _tts.setSpeechRate(rate);
    // speak가 다 읽을 때까지 기다리게 합니다 (재생 큐가 다음 문장을 기다립니다).
    await _tts.awaitSpeakCompletion(true);
  }

  @override
  Future<void> speak(String text) async {
    await (_ready ??= _setUp());
    await _tts.speak(text);
  }

  @override
  Future<void> stop() async {
    await _tts.stop();
  }
}

/// 기기 음성 인식으로 한 마디씩 듣는 [SpeechRecognizerPort]
/// (안드로이드는 SpeechRecognizer, 대개 Google 음성 인식).
class SpeechToTextRecognizer implements SpeechRecognizerPort {
  SpeechToTextRecognizer({
    SpeechToText? speech,
    this.localeId = 'ko_KR',
    this.pauseFor = const Duration(seconds: 3),
    this.listenFor = const Duration(seconds: 30),
  }) : _speech = speech ?? SpeechToText();

  final SpeechToText _speech;

  /// 알아들을 말.
  final String localeId;

  /// 이만큼 조용하면 한 마디가 끝난 것으로 봅니다. 어르신은 말 중간에 쉬는
  /// 시간이 길어 넉넉히 둡니다.
  final Duration pauseFor;

  /// 한 마디의 최대 길이.
  final Duration listenFor;

  bool _ready = false;
  Completer<String?>? _result;
  void Function()? _onSpeech;
  var _words = '';

  // 말이 없어서 끝난 경우. 오류가 아니라 다시 들으면 됩니다.
  static const _quietErrors = {'error_no_match', 'error_speech_timeout'};

  @override
  Future<String?> listen({
    void Function()? onSpeech,
    void Function(double level)? onLevel,
  }) async {
    if (!_ready) {
      // 처음 한 번 마이크 권한을 묻습니다. 실패하면 다음에 다시 해 봅니다.
      _ready = await _speech.initialize(onStatus: _onStatus, onError: _onError);
      if (!_ready) throw StateError('이 기기에서 음성 인식을 쓸 수 없습니다');
    }
    final result = _result = Completer<String?>();
    _words = '';
    _onSpeech = onSpeech;
    await _speech.listen(
      onResult: _onResult,
      onSoundLevelChange: onLevel == null
          ? null
          : (db) => onLevel(soundLevel(db)),
      listenOptions: SpeechListenOptions(
        localeId: localeId,
        pauseFor: pauseFor,
        listenFor: listenFor,
        listenMode: ListenMode.dictation,
        cancelOnError: true,
      ),
    );
    return result.future;
  }

  @override
  Future<void> cancel() async {
    _finish(null);
    if (_ready) await _speech.cancel();
  }

  /// 안드로이드 음성 인식이 주는 소리 크기(대략 -2~10 dB)를 0~1로 바꿉니다.
  static double soundLevel(double db) => ((db + 2) / 12).clamp(0, 1);

  void _onResult(SpeechRecognitionResult result) {
    final words = result.recognizedWords;
    if (_words.isEmpty && words.isNotEmpty) _onSpeech?.call();
    _words = words;
    if (result.finalResult) _finish(words);
  }

  void _onStatus(String status) {
    // 마지막 결과 없이 끝나면 들은 데까지 씁니다.
    if (status == SpeechToText.doneStatus) _finish(_words);
  }

  void _onError(SpeechRecognitionError error) {
    if (_quietErrors.contains(error.errorMsg)) {
      _finish(null);
      return;
    }
    final result = _result;
    _result = null;
    if (result != null && !result.isCompleted) {
      result.completeError(StateError('음성 인식 오류: ${error.errorMsg}'));
    }
  }

  void _finish(String? words) {
    final result = _result;
    _result = null;
    if (result != null && !result.isCompleted) {
      result.complete(words == null || words.isEmpty ? null : words);
    }
  }
}
