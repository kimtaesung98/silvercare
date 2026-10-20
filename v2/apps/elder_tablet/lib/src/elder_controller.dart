import 'dart:async';
import 'dart:collection';

import 'package:flutter/foundation.dart';

import 'chat_client.dart';
import 'voice.dart';

/// 어르신 화면의 마이크 원 상태. 피그마 컴포넌트 `표시/마이크`의 상태와 같습니다.
enum MicState { idle, listening, speaking }

/// 어르신 화면 뒤에서 도는 대화 흐름입니다.
///
/// 듣기 → 들은 말을 방에 보냄 → 방에 온 다른 참가자(말동무, 안전 지킴이)의 말을 소리로 읽기
/// → 다시 듣기를 되풀이합니다. 위급한 말은 서버의 위급 필터가 말동무보다 먼저 보므로
/// 태블릿은 들은 말을 그대로 보내기만 합니다.
class ElderController extends ChangeNotifier {
  ElderController({
    required this.chat,
    required this.voice,
    required this.elderName,
    this.retryDelay = const Duration(seconds: 2),
    this.replyWait = const Duration(seconds: 12),
  });

  final Chat chat;
  final Voice voice;
  final String elderName;

  /// 연결이 끊겼을 때, 아무 말도 못 들었을 때 다시 해 보기 전에 쉬는 시간.
  final Duration retryDelay;

  /// 말을 보낸 뒤 대답을 기다리는 동안 "듣고 있어요"를 유지하는 시간.
  final Duration replyWait;

  MicState _state = MicState.idle;
  MicState get state => _state;

  /// 서버와 이어지지 않을 때 true. 화면은 "잠시만요, 다시 이어 볼게요"를 보여 줍니다.
  bool get offline => _offline;
  bool _offline = false;

  Joined? _me;
  final _joined = Completer<void>();
  bool _running = false;
  int _lastSeq = 0;
  bool _awaitingReply = false;
  Timer? _replyTimer;
  final _toSpeak = Queue<String>();
  StreamSubscription<ChatMessage>? _sub;

  /// 방을 열고 듣기 시작합니다.
  Future<void> start() async {
    if (_running) return;
    _running = true;
    unawaited(_connect());
    unawaited(_loop());
  }

  /// 듣기와 연결을 멈춥니다 (보호자 화면으로 돌아갈 때).
  Future<void> stop() async {
    _running = false;
    _replyTimer?.cancel();
    await _sub?.cancel();
    _sub = null;
    await voice.stopListening();
  }

  @override
  void dispose() {
    _running = false;
    _replyTimer?.cancel();
    _sub?.cancel();
    super.dispose();
  }

  void _set(MicState s) {
    if (_state == s) return;
    _state = s;
    notifyListeners();
  }

  void _setOffline(bool v) {
    if (_offline == v) return;
    _offline = v;
    notifyListeners();
  }

  /// 방을 열고 메시지 흐름을 받습니다. 끊기면 잠깐 쉬고 다시 붙고, 이미 받은 메시지는 건너뜁니다.
  Future<void> _connect() async {
    while (_running) {
      try {
        _me ??= await chat.openRoom(elderName);
        if (!_joined.isCompleted) _joined.complete();
        final done = Completer<void>();
        _sub = chat
            .events(_me!.roomId)
            .listen(
              _onMessage,
              onError: (Object _) => done.isCompleted ? null : done.complete(),
              onDone: () => done.isCompleted ? null : done.complete(),
            );
        _setOffline(false);
        await done.future;
      } catch (_) {
        // 아래에서 쉬고 다시 해 봅니다.
      }
      if (!_running) return;
      _setOffline(true);
      await Future<void>.delayed(retryDelay);
    }
  }

  void _onMessage(ChatMessage m) {
    if (m.seq <= _lastSeq) return;
    _lastSeq = m.seq;
    if (m.fromId == _me?.meId) return;
    // 말동무, 안전 지킴이, 조무사의 말을 소리로 읽어 드립니다. 글로는 띄우지 않습니다.
    _toSpeak.add(m.text);
    _awaitingReply = false;
    _replyTimer?.cancel();
    if (_state != MicState.speaking) voice.stopListening();
  }

  Future<void> _loop() async {
    // 방이 열리기 전에는 듣지 않습니다. 들은 말을 보낼 곳이 없기 때문입니다.
    await _joined.future;
    while (_running) {
      if (_toSpeak.isNotEmpty) {
        _set(MicState.speaking);
        while (_toSpeak.isNotEmpty && _running) {
          await voice.speak(_toSpeak.removeFirst());
        }
        continue;
      }
      _set(_awaitingReply ? MicState.listening : MicState.idle);
      final text = await voice.listen(onHeard: () => _set(MicState.listening));
      if (!_running) return;
      if (text == null) {
        if (_toSpeak.isEmpty) await Future<void>.delayed(retryDelay ~/ 4);
        continue;
      }
      await _send(text);
    }
  }

  Future<void> _send(String text) async {
    final me = _me;
    if (me == null) return;
    try {
      await chat.post(me, text);
      if (_toSpeak.isNotEmpty) return; // 대답이 이미 왔음
      _awaitingReply = true;
      _replyTimer?.cancel();
      _replyTimer = Timer(replyWait, () {
        _awaitingReply = false;
        if (_state == MicState.listening) _set(MicState.idle);
      });
    } catch (_) {
      _setOffline(true);
    }
  }
}
