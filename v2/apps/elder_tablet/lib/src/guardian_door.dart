import 'package:flutter/gestures.dart';
import 'package:flutter/widgets.dart';

/// 어르신 화면에서 보호자 화면으로 돌아가는 문입니다.
///
/// 임시 방식: 시계를 [hold] 동안 누릅니다 (피그마 "켜 두기 준비" 화면의 안내 문구).
/// 어떻게 돌아갈지는 명세에서 아직 정하는 중이라 (PR #29 열린 질문 2), 바꿀 때는
/// 이 위젯 하나만 고칩니다. 예: 네 귀퉁이 차례로 누르기, 보호자 비밀번호.
class GuardianDoor extends StatelessWidget {
  const GuardianDoor({super.key, required this.onOpen, required this.child});

  final VoidCallback onOpen;
  final Widget child;

  static const hold = Duration(seconds: 3);

  @override
  Widget build(BuildContext context) {
    return RawGestureDetector(
      behavior: HitTestBehavior.opaque,
      gestures: {
        LongPressGestureRecognizer: GestureRecognizerFactoryWithHandlers<LongPressGestureRecognizer>(
          () => LongPressGestureRecognizer(duration: hold),
          (r) => r.onLongPress = onOpen,
        ),
      },
      child: child,
    );
  }
}
