import 'package:caregiver/main.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('첫 화면이 앱 이름을 보여준다', (tester) async {
    await tester.pumpWidget(const CaregiverApp());
    expect(find.text('노치원 조무사'), findsOneWidget);
  });
}
