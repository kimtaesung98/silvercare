import 'dart:async';

import 'package:api_client/api_client.dart';
import 'package:flutter/foundation.dart';
import 'package:ui/ui.dart';
import 'package:voice/voice.dart';

/// 화면에 보여줄 상태.
enum ScreenState {
  /// 대기 화면: 세션이 없습니다. 말동무 버튼만 보입니다.
  idle,

  /// 대화 화면.
  talking,

  /// 도착 임박: 조무사 ETA가 2분 이하입니다.
  arriving,
}

/// 태블릿이 켜질 때 받아두는 기기·어르신 정보 (`GET /tablet/context`).
class TabletInfo {
  const TabletInfo({required this.elderName, required this.companionEnabled});

  /// 어르신 이름.
  final String elderName;

  /// 말동무 버튼을 보여줄지.
  final bool companionEnabled;
}

/// 태블릿의 대화 상태를 모읍니다: WebSocket 이벤트를 받아 재생 큐에 넣고,
/// 마이크가 잡은 발화를 서버로 보냅니다.
///
/// 화면은 이 객체만 보고 그립니다.
class SessionController extends ChangeNotifier {
  SessionController({
    required this.socket,
    required this.queue,
    required this.clips,
    required this.recorder,
    this.loadInfo,
    this.arrivingMinutes = 2,
    this.muteWhileSpeaking = true,
  });

  /// `/ws/elder` 연결.
  final ElderSocket socket;

  /// 0번 → 1번 → 2번 재생 큐.
  final PlaybackQueue queue;

  /// 0번 문장 음성 캐시.
  final ClipCache clips;

  /// 마이크.
  final UtteranceRecorder recorder;

  /// 기기·어르신 정보를 받아오는 통로 (없으면 기본 호칭을 씁니다).
  final Future<TabletInfo> Function()? loadInfo;

  /// 이 분 이하로 남으면 도착 임박 화면으로 바꿉니다.
  final int arrivingMinutes;

  /// AI가 말하는 동안 마이크 입력을 무시할지.
  ///
  /// 태블릿 스피커 소리가 마이크로 되돌아오면 끼어들기가 잘못 잡혀서 기본은
  /// 켜 둡니다. 에코 제거가 확실한 기기에서는 끄면 말하는 중에도 끼어들 수
  /// 있습니다 (architecture.md 결정 22).
  final bool muteWhileSpeaking;

  final _subs = <StreamSubscription<dynamic>>[];
  final _captions = <CaptionLine>[];

  String? _sessionId;
  SessionMode? _mode;
  int? _etaMinutes;
  int _turnId = 0;
  int _playedIndex = 0;
  bool _connected = false;
  TabletInfo? _info;
  String? _notice;
  var _clientSeq = 0;

  // 측정값 (development-process.md 단계 4).
  DateTime? _utteranceEnd;
  DateTime? _openerEnd;
  int? _utteranceEndToOpenerMs;
  int? _openerEndToFirstReplyMs;

  /// 지금 화면.
  ScreenState get screen {
    if (_sessionId == null) return ScreenState.idle;
    final eta = _etaMinutes;
    if (_mode == SessionMode.PICKUP_BRIDGE &&
        eta != null &&
        eta <= arrivingMinutes) {
      return ScreenState.arriving;
    }
    return ScreenState.talking;
  }

  String? get sessionId => _sessionId;

  SessionMode? get mode => _mode;

  int? get etaMinutes => _etaMinutes;

  bool get connected => _connected;

  /// 기기·어르신 정보 (아직 못 받았으면 null).
  TabletInfo? get info => _info;

  /// 어르신 호칭.
  String get elderName {
    final name = _info?.elderName;
    return name == null ? '어르신' : '$name 어르신';
  }

  /// 말동무 버튼을 보여줄지 (서버가 끄면 숨깁니다).
  bool get companionEnabled => _info?.companionEnabled ?? true;

  /// AI가 말하는 중인지 (아니면 듣는 중).
  bool get speaking => queue.isBusy;

  /// 어르신에게 보여줄 안내 (거절 사유, 오류).
  String? get notice => _notice;

  /// 최근 대화 몇 줄 (새 줄이 아래).
  List<CaptionLine> get captions => List.unmodifiable(_captions);

  /// 이벤트를 받기 시작합니다.
  Future<void> start() async {
    _subs
      ..add(socket.events.listen(_onEvent))
      ..add(socket.states.listen(_onState))
      ..add(queue.started.listen(_onPlaybackStarted))
      ..add(queue.finished.listen(_onPlaybackFinished))
      ..add(recorder.utterances.listen(_onUtterance))
      ..add(recorder.speechStarted.listen((_) => _onSpeechStarted()));
    unawaited(_loadTabletInfo());
    await socket.start();
  }

  /// 말동무 버튼.
  void requestCompanion() {
    _notice = null;
    socket.send(ElderCommand.requestCompanion(), id: _nextId());
    notifyListeners();
  }

  /// 그만하기 버튼.
  void endSession() {
    final id = _sessionId;
    if (id != null) socket.send(ElderCommand.endSession(id), id: _nextId());
  }

  /// 개발용 텍스트 발화.
  void sendText(String text) {
    final id = _sessionId;
    if (id == null || text.trim().isEmpty) return;
    socket.send(
      ElderCommand.text(sessionId: id, clientId: _nextId(), text: text),
      id: _nextId(),
    );
  }

  @override
  void dispose() {
    for (final s in _subs) {
      unawaited(s.cancel());
    }
    unawaited(recorder.close());
    unawaited(queue.close());
    unawaited(socket.close());
    super.dispose();
  }

  Future<void> _loadTabletInfo() async {
    final load = loadInfo;
    if (load == null) return;
    try {
      _info = await load();
      notifyListeners();
    } on Object catch (e) {
      debugPrint('기기 정보 받기 실패: $e');
    }
  }

  String _nextId() => 'c-${++_clientSeq}';

  void _onState(ConnectionState state) {
    _connected = state == ConnectionState.connected;
    notifyListeners();
  }

  Future<void> _onEvent(ElderEvent event) async {
    switch (event) {
      case ConnectionReady(:final activeSession, :final openerVersion):
        unawaited(_syncClips(openerVersion));
        if (activeSession != null) {
          _enterSession(activeSession.sessionId, activeSession.mode, null);
        } else {
          await _leaveSession();
        }
      case SessionStarted(
        :final sessionId,
        :final mode,
        :final caregiverEtaMinutes,
      ):
        _captions.clear();
        _enterSession(sessionId, mode, caregiverEtaMinutes);
      case SessionRejected(:final reason):
        _notice = _rejectionText(reason);
        notifyListeners();
      case SessionEnded():
        await _leaveSession();
      case CaregiverEta(:final minutes):
        _etaMinutes = minutes;
        notifyListeners();
      case ElderTranscript(:final text):
        _addCaption(CaptionLine(text: text, isElder: true));
      case AiOpener(:final turnId, :final clipId):
        _turnId = turnId;
        _playedIndex = 0;
        if (muteWhileSpeaking) recorder.muted = true;
        queue.add(
          PlayItem.clip(
            turnId: turnId,
            index: 0,
            clipId: clipId,
            text: clips.text(clipId),
          ),
        );
      case AiFiller(:final turnId, :final clipId):
        queue.add(PlayItem.clip(turnId: turnId, index: -1, clipId: clipId));
      case AiReply(:final turnId, :final index, :final text, :final audio):
        _turnId = turnId;
        if (muteWhileSpeaking) recorder.muted = true;
        queue.add(
          PlayItem(turnId: turnId, index: index, audio: audio, text: text),
        );
      case AiTurnEnd(:final turnId):
        _sendMetrics(turnId);
      case ServerError(:final code, :final message):
        if (code == 'STT_FAILED') {
          _notice = '잘 못 들었어요. 한 번만 더 말씀해 주세요.';
        } else if (code == 'INTERNAL') {
          _notice = '잠시 문제가 있어요. 금방 다시 해볼게요.';
        }
        debugPrint('서버 오류 $code: $message');
        notifyListeners();
      case UnknownEvent():
      case Ping():
      case Pong():
        break;
    }
  }

  void _enterSession(String id, SessionMode mode, int? eta) {
    _sessionId = id;
    _mode = mode;
    _etaMinutes = eta;
    _notice = null;
    unawaited(recorder.start());
    notifyListeners();
  }

  Future<void> _leaveSession() async {
    _sessionId = null;
    _mode = null;
    _etaMinutes = null;
    await queue.clear();
    await recorder.stop();
    notifyListeners();
  }

  Future<void> _syncClips(String version) async {
    try {
      await clips.sync(serverVersion: version);
    } on Object catch (e) {
      debugPrint('0번 문장 캐시 갱신 실패: $e');
    }
  }

  void _onPlaybackStarted(PlayItem item) {
    if (item.index >= 0) _playedIndex = item.index;
    final now = DateTime.now();
    if (item.index == 0 && _utteranceEnd != null) {
      _utteranceEndToOpenerMs = now.difference(_utteranceEnd!).inMilliseconds;
    }
    if (item.index == 1 && _openerEnd != null) {
      _openerEndToFirstReplyMs = now.difference(_openerEnd!).inMilliseconds;
    }
    final text = item.text;
    if (text != null) _addCaption(CaptionLine(text: text, isElder: false));
  }

  void _onPlaybackFinished(PlayItem item) {
    if (item.index == 0) _openerEnd = DateTime.now();
    if (muteWhileSpeaking && !queue.isBusy) {
      // AI가 말을 멈췄으니 다시 듣습니다.
      recorder.muted = false;
    }
    notifyListeners();
  }

  /// 어르신이 재생 중에 말을 시작하면 끼어들기입니다.
  void _onSpeechStarted() {
    if (!queue.isBusy) return;
    final id = _sessionId;
    if (id == null) return;
    socket.send(
      ElderCommand.bargeIn(
        sessionId: id,
        turnId: _turnId,
        playedIndex: _playedIndex,
      ),
    );
    unawaited(queue.cancelTurn(_turnId));
  }

  void _onUtterance(Uint8List pcm) {
    final id = _sessionId;
    if (id == null) return;
    _utteranceEnd = DateTime.now();
    _utteranceEndToOpenerMs = null;
    _openerEndToFirstReplyMs = null;
    socket.send(
      ElderCommand.audio(sessionId: id, clientId: _nextId(), audio: pcm),
      id: _nextId(),
    );
  }

  void _sendMetrics(int turnId) {
    final id = _sessionId;
    final end = _utteranceEnd;
    if (id == null || end == null || _utteranceEndToOpenerMs == null) return;
    socket.send(
      ElderCommand.metrics(
        sessionId: id,
        turnId: turnId,
        utteranceEndToOpenerMs: _utteranceEndToOpenerMs,
        openerEndToFirstReplyMs: _openerEndToFirstReplyMs,
        utteranceEndToFirstReplyMs:
            _utteranceEndToOpenerMs == null || _openerEndToFirstReplyMs == null
            ? null
            : _utteranceEndToOpenerMs! + _openerEndToFirstReplyMs!,
      ),
    );
    _utteranceEnd = null;
  }

  void _addCaption(CaptionLine caption) {
    _captions.add(caption);
    if (_captions.length > 6) _captions.removeAt(0);
    notifyListeners();
  }

  static String _rejectionText(String reason) => switch (reason) {
    'ALREADY_ACTIVE' => '지금 이야기 중이에요.',
    'COMPANION_DISABLED' => '지금은 말동무를 쓸 수 없어요.',
    'BEDTIME' => '지금은 주무실 시간이에요. 내일 또 이야기해요.',
    'TOKEN_LIMIT' => '오늘은 이야기를 많이 나눴어요. 내일 또 해요.',
    _ => '지금은 시작할 수 없어요.',
  };
}
