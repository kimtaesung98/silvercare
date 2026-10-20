import 'dart:async';

import 'package:elder_tablet/src/chat_client.dart';
import 'package:elder_tablet/src/guardian_screen.dart';
import 'package:elder_tablet/src/voice.dart';

/// 서버 대신. 어르신이 보낸 말을 [posted]에 모으고, [reply]로 다른 참가자의 말을 흘려 보냅니다.
class FakeChat implements Chat {
  bool up = true;
  final posted = <String>[];
  final _events = StreamController<ChatMessage>.broadcast();
  int _seq = 0;
  String? openedFor;

  @override
  Future<bool> healthy() async => up;

  @override
  Future<Joined> openRoom(String elderName) async {
    openedFor = elderName;
    return const Joined('r1', 'h1');
  }

  @override
  Future<void> post(Joined me, String text) async {
    posted.add(text);
    _events.add(ChatMessage(seq: ++_seq, fromId: me.meId, fromName: '어르신', fromKind: 'human', text: text));
  }

  void reply(String text, {String kind = 'ai'}) {
    _events.add(ChatMessage(seq: ++_seq, fromId: 'ai-r1', fromName: '말동무', fromKind: kind, text: text));
  }

  @override
  Stream<ChatMessage> events(String roomId) => _events.stream;
}

/// 마이크와 스피커 대신. [hear]로 어르신이 말한 것처럼, [finishSpeaking]으로 읽기가 끝난 것처럼 합니다.
class FakeVoice implements Voice {
  bool micOk = true;
  final spoken = <String>[];
  Completer<String?>? _listening;
  void Function()? _onHeard;
  Completer<void>? _speaking;

  bool get isListening => _listening != null && !_listening!.isCompleted;

  @override
  Future<bool> init() async => micOk;

  @override
  Future<String?> listen({required void Function() onHeard}) {
    _onHeard = onHeard;
    return (_listening = Completer<String?>()).future;
  }

  /// 어르신이 말을 시작함 (부분 결과가 들어옴).
  void startHearing() => _onHeard?.call();

  /// 어르신이 말을 마침.
  void hear(String text) {
    _onHeard?.call();
    _listening?.complete(text);
  }

  @override
  Future<void> stopListening() async {
    final l = _listening;
    if (l != null && !l.isCompleted) l.complete(null);
  }

  @override
  Future<void> speak(String text) {
    spoken.add(text);
    return (_speaking = Completer<void>()).future;
  }

  void finishSpeaking() => _speaking?.complete();
}

class FakeCharger implements Charger {
  bool on = true;

  @override
  Future<bool> plugged() async => on;
}
