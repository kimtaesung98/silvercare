import 'package:elder_tablet/src/chat_client.dart';
import 'package:elder_tablet/src/elder_controller.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fakes.dart';

void main() {
  late FakeChat chat;
  late FakeVoice voice;
  late ElderController c;

  setUp(() {
    chat = FakeChat();
    voice = FakeVoice();
    c = ElderController(
      chat: chat,
      voice: voice,
      elderName: '김순자',
      retryDelay: const Duration(milliseconds: 20),
      replyWait: const Duration(milliseconds: 200),
    );
  });

  tearDown(() => c.stop());

  Future<void> settle() => Future<void>.delayed(const Duration(milliseconds: 10));

  test('기다림 → 듣는 중 → 말하는 중 → 기다림, 말동무 말은 소리로', () async {
    await c.start();
    await settle();
    expect(chat.openedFor, '김순자');
    expect(c.state, MicState.idle);

    voice.startHearing();
    expect(c.state, MicState.listening);

    voice.hear('안녕');
    await settle();
    expect(chat.posted, ['안녕']);
    expect(c.state, MicState.listening, reason: '대답을 기다리는 동안 듣는 중을 유지');

    chat.reply('안녕하세요, 반가워요.');
    await settle();
    expect(c.state, MicState.speaking);
    expect(voice.spoken, ['안녕하세요, 반가워요.'], reason: '내가 한 말은 읽지 않음');

    voice.finishSpeaking();
    await settle();
    expect(c.state, MicState.idle);
    expect(voice.isListening, isTrue);
  });

  test('위급한 말에 안전 지킴이가 보낸 안심 문장도 소리로 읽음', () async {
    await c.start();
    await settle();
    voice.hear('다리가 너무 아퍼');
    await settle();
    chat.reply('지금 많이 힘드시죠. 곧 도와드릴 분께 알렸어요.', kind: 'system');
    await settle();
    expect(voice.spoken.single, startsWith('지금 많이 힘드시죠'));
    expect(c.state, MicState.speaking);
  });

  test('대답이 늦으면 기다림으로 돌아감', () async {
    await c.start();
    await settle();
    voice.hear('여보세요');
    await settle();
    expect(c.state, MicState.listening);
    await Future<void>.delayed(const Duration(milliseconds: 250));
    expect(c.state, MicState.idle);
  });

  test('서버가 없으면 오프라인 표시 후 다시 이어짐', () async {
    final down = _DownChat();
    final c2 = ElderController(
      chat: down,
      voice: voice,
      elderName: '김순자',
      retryDelay: const Duration(milliseconds: 20),
    );
    await c2.start();
    await Future<void>.delayed(const Duration(milliseconds: 30));
    expect(c2.offline, isTrue);
    down.up = true;
    await Future<void>.delayed(const Duration(milliseconds: 60));
    expect(c2.offline, isFalse);
    await c2.stop();
  });
}

class _DownChat extends FakeChat {
  _DownChat() {
    up = false;
  }

  @override
  Future<Joined> openRoom(String elderName) async {
    if (!up) throw Exception('연결 안 됨');
    return super.openRoom(elderName);
  }
}
