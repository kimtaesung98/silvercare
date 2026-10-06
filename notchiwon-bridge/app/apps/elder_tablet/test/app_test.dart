import 'package:api_client/api_client.dart';
import 'package:elder_tablet/main.dart';
import 'package:elder_tablet/src/session_controller.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:voice/voice.dart';

import 'fakes.dart';

void main() {
  late FakeChannel channel;
  late FakeMicrophone mic;
  late SessionController controller;

  setUp(() async {
    channel = FakeChannel();
    mic = FakeMicrophone();
    controller = SessionController(
      socket: ElderSocket(
        serverUrl: Uri.parse('http://server'),
        token: 't',
        connect: (_, _) async => channel,
      ),
      queue: PlaybackQueue(FakePlayer()),
      clips: ClipCache(
        source: FakeClipSource(texts: {'clip-recall-1': '그러셨어요?'}),
        store: MemoryClipStore(),
      ),
      recorder: UtteranceRecorder(mic),
      loadInfo: () async =>
          const TabletInfo(elderName: '김영자', companionEnabled: true),
    );
    await controller.start();
  });

  tearDown(() => controller.dispose());

  /// 서버 프레임은 실제 비동기로 흘러야 컨트롤러에 닿습니다
  /// (위젯 테스트의 가짜 시간 안에서는 스트림이 전달되지 않습니다).
  Future<void> deliver(WidgetTester tester, void Function() send) async {
    await tester.runAsync(() async {
      send();
      await Future<void>.delayed(const Duration(milliseconds: 10));
    });
    await tester.pumpAndSettle();
  }

  Future<void> pumpApp(WidgetTester tester) async {
    // 태블릿 화면 (10인치 가로).
    tester.view
      ..physicalSize = const Size(2560, 1600)
      ..devicePixelRatio = 2.0;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(ElderTabletApp(controller: controller));
    await tester.pumpAndSettle();
  }

  testWidgets('대기 화면에 말동무 버튼이 보이고, 누르면 서버로 요청한다', (tester) async {
    await pumpApp(tester);

    expect(find.text('김영자 어르신'), findsOneWidget);
    expect(find.text('말동무 하기'), findsOneWidget);

    await tester.tap(find.text('말동무 하기'));
    await tester.pump();

    expect(channel.lastSent('session.request'), isNotNull);
  });

  testWidgets('세션이 시작되면 대화 화면에 자막과 그만하기 버튼이 보인다', (tester) async {
    await pumpApp(tester);
    await deliver(tester, () {
      channel
        ..serverSendsEvent('session.started', {
          'sessionId': 's-1',
          'mode': 'PICKUP_BRIDGE',
          'startedBy': 'CAREGIVER',
          'caregiverEtaMinutes': 10,
        })
        ..serverSendsEvent('ai.reply', {
          'sessionId': 's-1',
          'turnId': 1,
          'index': 1,
          'utteranceId': 'u-1',
          'text': '오늘 날씨가 좋네요.',
        });
    });

    expect(find.text('오늘 날씨가 좋네요.'), findsOneWidget);
    expect(find.text('그만하기'), findsOneWidget);
    expect(find.text('10분 뒤 도착'), findsOneWidget);
  });

  testWidgets('조무사가 거의 다 오면 도착 임박 화면이 가장 크게 뜬다', (tester) async {
    await pumpApp(tester);
    await deliver(
      tester,
      () => channel.serverSendsEvent('session.started', {
        'sessionId': 's-1',
        'mode': 'PICKUP_BRIDGE',
        'startedBy': 'CAREGIVER',
        'caregiverEtaMinutes': 1,
      }),
    );

    expect(find.text('선생님이 1분 뒤에 와요'), findsOneWidget);
    expect(find.text('천천히 준비하세요'), findsOneWidget);
  });

  testWidgets('말동무를 쓸 수 없는 어르신에게는 버튼을 눌리지 않게 둔다', (tester) async {
    await pumpApp(tester);
    await deliver(
      tester,
      () => channel.serverSendsEvent('session.rejected', {
        'reason': 'COMPANION_DISABLED',
      }),
    );

    expect(find.text('지금은 말동무를 쓸 수 없어요.'), findsOneWidget);
  });
}
