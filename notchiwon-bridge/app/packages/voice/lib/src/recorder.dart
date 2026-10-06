import 'dart:async';
import 'dart:math' as math;
import 'dart:typed_data';

/// 마이크에서 PCM 조각을 내보내는 장치. 실제 구현은 `record` 패키지를 쓰고,
/// 테스트는 가짜를 씁니다.
abstract class MicrophonePort {
  /// 녹음을 시작하고 16kHz 모노 PCM16 조각을 내보냅니다.
  Future<Stream<Uint8List>> start();

  /// 녹음을 멈춥니다.
  Future<void> stop();
}

/// 발화 한 구간을 자르는 규칙 (architecture.md 6.1절 1번).
class SegmenterConfig {
  const SegmenterConfig({
    this.startThreshold = 0.08,
    this.endThreshold = 0.05,
    this.silenceToEnd = const Duration(milliseconds: 900),
    this.minUtterance = const Duration(milliseconds: 400),
    this.maxUtterance = const Duration(seconds: 30),
  });

  /// 이 세기를 넘으면 말이 시작된 것으로 봅니다 (0~1).
  final double startThreshold;

  /// 이 세기 아래면 조용한 것으로 봅니다. 시작보다 낮게 둬서 말끝이 잘리지
  /// 않게 합니다.
  final double endThreshold;

  /// 이만큼 조용하면 발화가 끝난 것으로 봅니다. 어르신은 말 중간에 쉬는
  /// 시간이 길어 넉넉히 둡니다.
  final Duration silenceToEnd;

  /// 이보다 짧으면 기침이나 잡음으로 보고 버립니다.
  final Duration minUtterance;

  /// 이보다 길면 거기서 한 번 끊습니다.
  final Duration maxUtterance;
}

/// 발화가 시작·끝났음을 알려주는 신호.
enum SpeechEvent { started, ended, tooShort }

/// PCM 조각의 세기를 보고 발화 구간을 찾습니다. 순수 Dart라 테스트하기 쉽고,
/// 마이크 플러그인과 떨어져 있습니다.
class SpeechSegmenter {
  SpeechSegmenter({this.config = const SegmenterConfig()});

  final SegmenterConfig config;
  final _chunks = <Uint8List>[];

  Duration _elapsed = Duration.zero;
  Duration? _speechStart;
  Duration? _silenceStart;

  /// 말하는 중인지.
  bool get isSpeaking => _speechStart != null;

  /// 지금까지 모은 발화.
  Uint8List get buffered {
    final total = _chunks.fold(0, (n, c) => n + c.length);
    final out = Uint8List(total);
    var at = 0;
    for (final c in _chunks) {
      out.setRange(at, at + c.length, c);
      at += c.length;
    }
    return out;
  }

  /// PCM 조각 하나를 넣고 일어난 일을 돌려줍니다.
  /// 16kHz 모노 PCM16 기준으로 길이를 시간으로 셉니다.
  SpeechEvent? add(Uint8List chunk, {int sampleRate = 16000}) {
    final level = rms(chunk);
    _elapsed += Duration(
      microseconds: (chunk.length / 2 / sampleRate * 1000000).round(),
    );

    if (!isSpeaking) {
      if (level < config.startThreshold) return null;
      _speechStart = _elapsed;
      _silenceStart = null;
      _chunks
        ..clear()
        ..add(chunk);
      return SpeechEvent.started;
    }

    _chunks.add(chunk);
    final spoken = _elapsed - _speechStart!;
    if (spoken >= config.maxUtterance) return _end();

    if (level >= config.endThreshold) {
      _silenceStart = null;
      return null;
    }
    _silenceStart ??= _elapsed;
    if (_elapsed - _silenceStart! < config.silenceToEnd) return null;
    return _end();
  }

  SpeechEvent _end() {
    final spoken = _elapsed - _speechStart!;
    final silence = _silenceStart == null
        ? Duration.zero
        : _elapsed - _silenceStart!;
    _speechStart = null;
    _silenceStart = null;
    if (spoken - silence < config.minUtterance) {
      _chunks.clear();
      return SpeechEvent.tooShort;
    }
    return SpeechEvent.ended;
  }

  /// 모은 발화를 비웁니다 (보낸 뒤에 부릅니다).
  void reset() {
    _chunks.clear();
    _speechStart = null;
    _silenceStart = null;
  }

  /// PCM16 조각의 세기 (0~1).
  static double rms(Uint8List pcm) {
    if (pcm.length < 2) return 0;
    final samples = pcm.buffer.asInt16List(
      pcm.offsetInBytes,
      pcm.lengthInBytes ~/ 2,
    );
    var sum = 0.0;
    for (final s in samples) {
      final v = s / 32768.0;
      sum += v * v;
    }
    return math.sqrt(sum / samples.length);
  }
}

/// 마이크를 열고 발화 한 구간이 끝날 때마다 PCM을 내보냅니다.
///
/// 말동무 모드에서는 대화 중에만 마이크를 엽니다 (집 안 사생활 보호,
/// architecture.md 6.4절).
class UtteranceRecorder {
  UtteranceRecorder(this._mic, {SegmenterConfig? config})
    : _segmenter = SpeechSegmenter(config: config ?? const SegmenterConfig());

  final MicrophonePort _mic;
  final SpeechSegmenter _segmenter;
  final _utterances = StreamController<Uint8List>.broadcast();
  final _speechStarted = StreamController<void>.broadcast();

  StreamSubscription<Uint8List>? _sub;
  bool _muted = false;

  /// 끝난 발화 한 구간.
  Stream<Uint8List> get utterances => _utterances.stream;

  /// 어르신이 말을 시작했다는 신호 (끼어들기 판단에 씁니다).
  Stream<void> get speechStarted => _speechStarted.stream;

  bool get isRecording => _sub != null;

  /// 마이크를 엽니다.
  Future<void> start() async {
    if (_sub != null) return;
    final stream = await _mic.start();
    _sub = stream.listen(_onChunk);
  }

  /// 마이크를 닫습니다.
  Future<void> stop() async {
    await _sub?.cancel();
    _sub = null;
    _segmenter.reset();
    await _mic.stop();
  }

  /// AI가 말하는 동안 마이크 입력을 무시할지. 스피커 소리가 마이크로 들어오는
  /// 태블릿에서 끼어들기가 잘못 잡히는 것을 막습니다.
  set muted(bool value) {
    _muted = value;
    if (value) _segmenter.reset();
  }

  bool get muted => _muted;

  Future<void> close() async {
    await stop();
    await _utterances.close();
    await _speechStarted.close();
  }

  void _onChunk(Uint8List chunk) {
    if (_muted) return;
    switch (_segmenter.add(chunk)) {
      case SpeechEvent.started:
        if (!_speechStarted.isClosed) _speechStarted.add(null);
      case SpeechEvent.ended:
        final pcm = _segmenter.buffered;
        _segmenter.reset();
        if (!_utterances.isClosed) _utterances.add(pcm);
      case SpeechEvent.tooShort:
      case null:
        break;
    }
  }
}
