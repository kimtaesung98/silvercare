import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:guardian/src/app.dart';
import 'package:guardian/src/push.dart';

import 'fakes.dart';

void main() {
  late FakeBackend backend;
  late FakePush push;

  setUp(() {
    backend = FakeBackend()
      ..digestList.addAll([
        digest(flag: '평소보다 말씀이 많으심'),
        digest(
          date: DateTime.utc(2026, 10, 5),
          summary: '조용히 지내셨어요.',
          sessions: 0,
        ),
      ]);
    push = FakePush();
  });

  Future<void> open(WidgetTester tester, {bool signedIn = true}) async {
    await tester.pumpWidget(
      ProviderScope(
        retry: (_, _) => null,
        overrides: overrides(backend: backend, push: push, signedIn: signedIn),
        child: const GuardianApp(),
      ),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('로그인하면 어르신 목록과 푸시 토큰 등록', (tester) async {
    await open(tester, signedIn: false);
    expect(find.text('로그인'), findsWidgets);

    await tester.enterText(find.byType(TextField).at(0), 'kim');
    await tester.enterText(find.byType(TextField).at(1), 'wrong');
    await tester.tap(find.widgetWithText(FilledButton, '로그인'));
    await tester.pumpAndSettle();
    expect(find.text('아이디나 비밀번호가 맞지 않아요'), findsOneWidget);

    await tester.enterText(find.byType(TextField).at(1), 'pw');
    await tester.tap(find.widgetWithText(FilledButton, '로그인'));
    await tester.pumpAndSettle();
    expect(find.text('김보호님'), findsOneWidget);
    expect(find.text('김순자 어르신'), findsOneWidget);
    expect(backend.pushTokens, ['fcm-token']);
  });

  testWidgets('어르신을 누르면 하루 소식이 날짜순으로 보인다', (tester) async {
    await open(tester);
    await tester.tap(find.text('김순자 어르신'));
    await tester.pumpAndSettle();

    expect(find.text('큰아들 이야기를 즐겁게 하셨어요.'), findsOneWidget);
    expect(find.text('평소보다 말씀이 많으심'), findsOneWidget);
    expect(find.text('조용히 지내셨어요.'), findsOneWidget);
    expect(find.text('대화 없음'), findsOneWidget);
    // 말동무가 켜져 있으면 안부 시각을 위에 알려줍니다.
    expect(find.text('말동무 켜짐'), findsOneWidget);
    expect(find.text('안부 전화 10:00, 15:30'), findsOneWidget);
  });

  testWidgets('하루 소식이 없으면 그렇다고 알려준다', (tester) async {
    backend.digestList.clear();
    await open(tester);
    await tester.tap(find.text('김순자 어르신'));
    await tester.pumpAndSettle();
    expect(find.text('아직 하루 소식이 없어요'), findsOneWidget);
  });

  testWidgets('위급 알림은 홈 맨 위에 뜨고 확인하면 사라진다', (tester) async {
    backend.escalations['x1'] = escalation();
    await open(tester);
    expect(find.textContaining('통증 호소'), findsOneWidget);

    await tester.tap(find.text('확인하기'));
    await tester.pumpAndSettle();
    expect(find.text('“다리가 아파”'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, '확인했어요'));
    await tester.pumpAndSettle();
    expect(find.textContaining('확인함'), findsOneWidget);

    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.text('확인하기'), findsNothing);
  });

  testWidgets('앱을 보는 중 온 위급 알림은 띠로 보여주고 누르면 열린다', (tester) async {
    backend.escalations['x1'] = escalation();
    await open(tester);
    push.alert!(
      const Alert(
        type: 'escalation',
        title: '위급 알림: 김순자 어르신',
        body: '통증 호소: “다리가 아파”',
        escalationId: 'x1',
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(MaterialBanner), findsOneWidget);

    await tester.tap(find.widgetWithText(TextButton, '보기'));
    await tester.pumpAndSettle();
    expect(find.text('위급 알림'), findsWidgets);
  });

  testWidgets('하루 소식 알림을 누르면 그 어르신 화면이 열린다', (tester) async {
    await open(tester);
    push.open!(
      Alert(
        type: 'digest',
        title: '김순자 어르신 하루 소식',
        body: '큰아들 이야기를 즐겁게 하셨어요.',
        elderId: elder.id,
        date: '2026-10-06',
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('큰아들 이야기를 즐겁게 하셨어요.'), findsOneWidget);
  });

  testWidgets('말동무 설정을 고쳐 저장한다', (tester) async {
    await open(tester);
    await tester.tap(find.text('김순자 어르신'));
    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(Icons.settings_outlined));
    await tester.pumpAndSettle();

    expect(find.text('10:00, 15:30'), findsOneWidget);
    expect(find.text('21:00 ~ 07:00 에는 말을 걸지 않아요'), findsOneWidget);
    expect(find.text('센터 기본값'), findsWidgets);

    // 안부 시각 하나를 지우고 말동무를 끕니다.
    await tester.tap(
      find.descendant(
        of: find.widgetWithText(InputChip, '10:00'),
        matching: find.byIcon(Icons.close),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byType(Switch));
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(FilledButton, '저장하기'));
    await tester.pumpAndSettle();
    expect(backend.saved, hasLength(1));
    final s = backend.saved.single;
    expect(s.enabled, isFalse);
    expect(s.checkInTimes, ['15:30']);
    expect(s.bedtimeStart, '21:00');
    expect(s.timeZone, 'Asia/Seoul');
    expect(find.text('저장했어요'), findsOneWidget);
  });

  testWidgets('불러오지 못하면 다시 시도할 수 있다', (tester) async {
    backend.digestList.clear();
    backend.digestError = Exception('down');
    await open(tester);
    await tester.tap(find.text('김순자 어르신'));
    await tester.pumpAndSettle();
    expect(find.text('서버에 연결하지 못했어요'), findsOneWidget);

    backend.digestError = null;
    backend.digestList.add(digest());
    await tester.tap(find.widgetWithText(OutlinedButton, '다시 불러오기'));
    await tester.pumpAndSettle();
    expect(find.text('큰아들 이야기를 즐겁게 하셨어요.'), findsOneWidget);
  });

  testWidgets('로그아웃하면 로그인 화면으로 간다', (tester) async {
    await open(tester);
    await tester.tap(find.byType(PopupMenuButton<void>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('로그아웃'));
    await tester.pumpAndSettle();
    expect(find.widgetWithText(FilledButton, '로그인'), findsOneWidget);
  });
}
