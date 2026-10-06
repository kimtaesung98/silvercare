import 'package:elder_tablet/main.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('대기 화면이 앱 이름을 보여준다', (tester) async {
    await tester.pumpWidget(const ElderTabletApp());
    expect(find.text('노치원 말동무'), findsOneWidget);
  });
}
