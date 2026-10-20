import 'package:elder_tablet/main.dart';
import 'package:elder_tablet/src/guardian_door.dart';
import 'package:elder_tablet/src/elder_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fakes.dart';

void main() {
  late FakeChat chat;
  late FakeVoice voice;
  late FakeCharger charger;

  setUp(() {
    chat = FakeChat();
    voice = FakeVoice();
    charger = FakeCharger();
  });

  Future<void> pumpApp(WidgetTester t) async {
    t.view.physicalSize = const Size(1280, 800);
    t.view.devicePixelRatio = 1;
    addTearDown(t.view.reset);
    await t.pumpWidget(ElderTabletApp(chat: chat, voice: voice, charger: charger, kiosk: false));
    await t.pump();
  }

  testWidgets('보호자가 확인하고 켜 두면 어르신 화면, 시계 3초면 다시 보호자 화면', (t) async {
    await pumpApp(t);
    expect(find.text('말동무 켜 두기'), findsOneWidget);
    expect(find.text('충전기에 꽂혀 있어요'), findsOneWidget);
    expect(find.text('인터넷이 연결됐어요'), findsOneWidget);
    expect(find.text('마이크가 잘 들려요'), findsOneWidget);

    FilledButton button() => t.widget<FilledButton>(find.byKey(const Key('keep-on')));
    expect(button().onPressed, isNull, reason: '어르신 이름이 없으면 못 누름');
    await t.enterText(find.byKey(const Key('elder-name')), '김순자');
    await t.pump();
    expect(button().onPressed, isNotNull);

    await t.tap(find.byKey(const Key('keep-on')));
    await t.pump();
    await t.pump();
    expect(find.text('말을 걸어 주세요'), findsOneWidget);
    expect(chat.openedFor, '김순자');

    voice.hear('안녕');
    await t.pump();
    expect(find.text('듣고 있어요'), findsOneWidget);
    chat.reply('안녕하세요');
    await t.pump();
    await t.pump();
    expect(find.text('말동무가 대답해요'), findsOneWidget);
    expect(find.text('안녕하세요'), findsNothing, reason: '말동무 말은 글로 띄우지 않음');
    voice.finishSpeaking();
    await t.pump();
    await t.pump();
    expect(find.text('말을 걸어 주세요'), findsOneWidget);

    // 짧게 누르면 아무 일 없음
    final door = find.byType(GuardianDoor);
    final g = await t.startGesture(t.getCenter(door));
    await t.pump(const Duration(seconds: 1));
    await g.up();
    await t.pump();
    expect(find.byType(ElderScreen), findsOneWidget);

    final g2 = await t.startGesture(t.getCenter(door));
    await t.pump(GuardianDoor.hold + const Duration(milliseconds: 100));
    await g2.up();
    await t.pumpAndSettle();
    expect(find.text('말동무 켜 두기'), findsOneWidget);
  });

  testWidgets('서버나 마이크가 안 되면 켜 둘 수 없음', (t) async {
    chat.up = false;
    voice.micOk = false;
    charger.on = false;
    await pumpApp(t);
    expect(find.text('충전기를 꽂아 주세요'), findsOneWidget);
    expect(find.text('서버에 연결되지 않아요'), findsOneWidget);
    expect(find.text('마이크를 허용해 주세요'), findsOneWidget);
    await t.enterText(find.byKey(const Key('elder-name')), '김순자');
    await t.pump();
    expect(t.widget<FilledButton>(find.byKey(const Key('keep-on'))).onPressed, isNull);

    chat.up = true;
    voice.micOk = true;
    await t.tap(find.text('다시 확인'));
    await t.pump();
    await t.pump();
    expect(t.widget<FilledButton>(find.byKey(const Key('keep-on'))).onPressed, isNotNull);
  });

  test('시계 글', () {
    expect(clockText(DateTime(2026, 10, 20, 15, 20)), '오후 3:20');
    expect(clockText(DateTime(2026, 10, 20, 0, 5)), '오전 12:05');
    expect(clockText(DateTime(2026, 10, 20, 12, 0)), '오후 12:00');
  });
}
