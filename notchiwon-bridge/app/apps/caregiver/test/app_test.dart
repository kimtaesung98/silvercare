import 'package:api_client/api_client.dart';
import 'package:caregiver/src/app.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fakes.dart';

void main() {
  late FakeBackend backend;
  late FakePositions positions;
  late FakePush push;

  setUp(() {
    backend = FakeBackend()
      ..visits['v1'] = visit()
      ..visits['v2'] = visit(id: 'v2', elder: '박영수', destination: null);
    backend.visits['v2']!.destination = null;
    positions = FakePositions();
    push = FakePush();
  });

  Future<void> open(WidgetTester tester, {bool signedIn = true}) async {
    await tester.pumpWidget(
      ProviderScope(
        retry: (_, _) => null,
        overrides: overrides(
          backend: backend,
          positions: positions,
          push: push,
          signedIn: signedIn,
        ),
        child: const CaregiverApp(),
      ),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('로그인하면 오늘 방문 목록과 푸시 토큰 등록', (tester) async {
    await open(tester, signedIn: false);
    expect(find.text('로그인'), findsWidgets);

    await tester.enterText(find.byType(TextField).at(0), 'lee');
    await tester.enterText(find.byType(TextField).at(1), 'wrong');
    await tester.tap(find.widgetWithText(FilledButton, '로그인'));
    await tester.pumpAndSettle();
    expect(find.text('아이디나 비밀번호가 맞지 않아요'), findsOneWidget);

    await tester.enterText(find.byType(TextField).at(1), 'pw');
    await tester.tap(find.widgetWithText(FilledButton, '로그인'));
    await tester.pumpAndSettle();
    expect(find.text('이조무 선생님의 오늘 방문'), findsOneWidget);
    expect(find.text('김순자 어르신'), findsOneWidget);
    expect(find.text('박영수 어르신'), findsOneWidget);
    expect(push.alert, isNotNull);
    expect(backend.pushTokens, ['fcm-token']);
  });

  testWidgets('출발하면 남은 시간과 대화 시작을 보여준다', (tester) async {
    await open(tester);
    await tester.tap(find.text('김순자 어르신'));
    await tester.pumpAndSettle();
    expect(find.text('서울 종로구 세종대로 175'), findsOneWidget);
    expect(find.text('카카오맵으로 길안내'), findsOneWidget);

    backend.onLocation = (id) => LocationUpdateResult(
      visitId: id,
      status: VisitStatus.SESSION_ACTIVE,
      etaMinutes: 9,
      sessionStarted: true,
    );
    await tester.tap(find.text('출발하기'));
    await tester.pumpAndSettle();
    positions.move(37.56, 126.97, now);
    await tester.pumpAndSettle();
    expect(find.text('도착까지 약 9분'), findsOneWidget);
    expect(find.text('어르신 태블릿에서 대화를 시작했어요'), findsOneWidget);
    expect(find.text('위치 보내기 멈추기'), findsOneWidget);

    await tester.tap(find.text('위치 보내기 멈추기'));
    await tester.pumpAndSettle();
    expect(find.text('출발하기'), findsOneWidget);
  });

  testWidgets('댁 위치가 없으면 지도 대신 안내', (tester) async {
    await open(tester);
    await tester.tap(find.text('박영수 어르신'));
    await tester.pumpAndSettle();
    expect(find.textContaining('위치가 등록되지 않았어요'), findsOneWidget);
    expect(find.text('카카오맵으로 길안내'), findsNothing);
  });

  testWidgets('도착하면 브리핑이 나올 때까지 기다렸다 보여주고 읽음 표시', (tester) async {
    backend
      ..briefingAfter = 2
      ..briefingValue = Briefing(
        sessionId: 's1',
        summaryText: '큰아들 이야기를 즐겁게 하심.',
        topKeywords: ['큰아들', '화투'],
        emotionFlag: '위급 감지: 통증 호소',
        overallEmotionTag: EmotionTag.UNUSUAL,
        escalationCount: 1,
        generatedAt: now,
      );
    await open(tester);
    await tester.tap(find.text('김순자 어르신'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('출발하기'));
    await tester.pumpAndSettle();

    await tester.tap(find.text('도착했어요'));
    await tester.pumpAndSettle();
    await tester.tap(
      find.descendant(
        of: find.byType(AlertDialog),
        matching: find.text('도착했어요'),
      ),
    );
    await tester.pump();
    await tester.pump();
    expect(backend.arrived, ['v1']);
    expect(find.text('브리핑을 만드는 중이에요…'), findsOneWidget);

    await tester.pump(const Duration(seconds: 3));
    await tester.pump(const Duration(seconds: 3));
    await tester.pumpAndSettle();
    expect(find.text('큰아들 이야기를 즐겁게 하심.'), findsOneWidget);
    expect(find.text('화투'), findsOneWidget);
    expect(find.textContaining('대화 중 위급 감지 1건'), findsOneWidget);
    expect(find.text('기분: 평소와 다름'), findsOneWidget);
    expect(backend.read, ['s1']);
    expect(find.text('출발하기'), findsNothing);
  });

  testWidgets('브리핑이 끝내 없으면 다시 열기 안내', (tester) async {
    backend.visits['v1'] = visit(
      status: VisitStatus.COMPLETED,
      sessionId: 's1',
    );
    await open(tester);
    await tester.tap(find.text('김순자 어르신'));
    for (var i = 0; i < 6; i++) {
      await tester.pump(const Duration(seconds: 3));
    }
    await tester.pumpAndSettle();
    expect(find.textContaining('잠시 뒤에 다시 열어 주세요'), findsOneWidget);
  });

  testWidgets('위급 알림을 확인하면 목록에서 빠진다', (tester) async {
    backend.escalations['x1'] = escalation();
    await open(tester);
    expect(find.text('김순자 어르신 · 통증 호소'), findsOneWidget);

    await tester.tap(find.text('김순자 어르신 · 통증 호소'));
    await tester.pumpAndSettle();
    expect(find.text('“다리가 아파”'), findsOneWidget);
    await tester.tap(find.text('확인했어요'));
    await tester.pumpAndSettle();
    expect(find.textContaining('에 확인함'), findsOneWidget);

    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.text('김순자 어르신 · 통증 호소'), findsNothing);
  });

  testWidgets('앱을 보는 중 온 위급 알림은 빨간 띠로, 누르면 자세히', (tester) async {
    await open(tester);
    backend.escalations['x1'] = escalation();
    push.alert!('x1', '위급 알림: 김순자 어르신', '통증 호소: “다리가 아파”');
    await tester.pumpAndSettle();
    expect(find.byType(MaterialBanner), findsOneWidget);
    expect(find.text('김순자 어르신 · 통증 호소'), findsOneWidget);

    await tester.tap(find.text('보기'));
    await tester.pumpAndSettle();
    expect(find.text('위급 알림'), findsOneWidget);
    expect(find.text('“다리가 아파”'), findsOneWidget);
  });

  testWidgets('알림을 눌러 앱을 열면 그 위급 알림으로 간다', (tester) async {
    backend.escalations['x1'] = escalation();
    await open(tester);
    push.open!('x1');
    await tester.pumpAndSettle();
    expect(find.text('“다리가 아파”'), findsOneWidget);
  });

  testWidgets('로그아웃하면 로그인 화면으로', (tester) async {
    await open(tester);
    await tester.tap(find.byType(PopupMenuButton<void>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('로그아웃'));
    await tester.pumpAndSettle();
    expect(find.widgetWithText(FilledButton, '로그인'), findsOneWidget);
  });
}
