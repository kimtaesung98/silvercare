import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui/ui.dart';

void main() {
  Future<void> pump(WidgetTester tester, Widget child) => tester.pumpWidget(
    MaterialApp(
      theme: elderTheme(),
      home: Scaffold(body: child),
    ),
  );

  testWidgets('BigButton은 어르신이 누를 수 있게 충분히 크다', (tester) async {
    var taps = 0;
    await pump(tester, BigButton(label: '말동무 하기', onPressed: () => taps++));

    final size = tester.getSize(find.byType(FilledButton));
    expect(size.height, greaterThanOrEqualTo(120));

    await tester.tap(find.text('말동무 하기'));
    expect(taps, 1);
  });

  testWidgets('onPressed가 없으면 눌리지 않는다', (tester) async {
    await pump(tester, const BigButton(label: '말동무 하기', onPressed: null));

    final button = tester.widget<FilledButton>(find.byType(FilledButton));
    expect(button.enabled, isFalse);
  });

  testWidgets('CaptionBoard는 최근 세 줄만 보여준다', (tester) async {
    await pump(
      tester,
      const CaptionBoard(
        lines: [
          CaptionLine(text: '첫째 줄', isElder: true),
          CaptionLine(text: '둘째 줄', isElder: false),
          CaptionLine(text: '셋째 줄', isElder: true),
          CaptionLine(text: '넷째 줄', isElder: false),
        ],
      ),
    );

    expect(find.text('첫째 줄'), findsNothing);
    expect(find.text('둘째 줄'), findsOneWidget);
    expect(find.text('넷째 줄'), findsOneWidget);
  });

  testWidgets('줄이 없으면 아무것도 그리지 않는다', (tester) async {
    await pump(tester, const CaptionBoard(lines: []));

    expect(find.byType(Text), findsNothing);
  });

  testWidgets('NoticeBar는 안내를 큰 글씨로 보여준다', (tester) async {
    await pump(tester, const NoticeBar(text: '지금은 주무실 시간이에요.'));

    expect(find.text('지금은 주무실 시간이에요.'), findsOneWidget);
  });
}
