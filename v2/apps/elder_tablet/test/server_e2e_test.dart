// 실제 v2 서버에 붙여 보는 시험입니다. 서버를 켜고 주소를 주면 돕니다:
//   (cd v2 && go run ./cmd/server) &
//   SILVERCARE_SERVER=http://localhost:8080 flutter test test/server_e2e_test.dart
import 'dart:io';

import 'package:elder_tablet/src/chat_client.dart';
import 'package:elder_tablet/src/elder_controller.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fakes.dart';

void main() {
  final server = Platform.environment['SILVERCARE_SERVER'];

  test('말을 걸면 말동무가 소리로 답하고, 위급한 말에는 안심 문장', () async {
    final chat = HttpChat(server!);
    final voice = FakeVoice();
    final c = ElderController(chat: chat, voice: voice, elderName: '김순자');
    await c.start();
    Future<void> waitFor(bool Function() ok) async {
      for (var i = 0; i < 100 && !ok(); i++) {
        await Future<void>.delayed(const Duration(milliseconds: 50));
      }
      expect(
        ok(),
        isTrue,
        reason: 'state=${c.state} offline=${c.offline} spoken=${voice.spoken} listening=${voice.isListening}',
      );
    }

    await waitFor(() => voice.isListening);
    voice.hear('안녕');
    await waitFor(() => voice.spoken.isNotEmpty);
    expect(c.state, MicState.speaking);
    voice.finishSpeaking();

    await waitFor(() => voice.isListening);
    voice.hear('다리가 너무 아퍼');
    await waitFor(() => voice.spoken.length == 2);
    expect(voice.spoken.last, contains('곧 도와드릴 분께 알렸어요'));
    voice.finishSpeaking();
    await c.stop();
  }, skip: server == null ? 'SILVERCARE_SERVER가 없어 건너뜀' : null);
}
