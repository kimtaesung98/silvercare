import 'package:flutter/material.dart';

import 'elder_theme.dart';

/// 어르신이 손가락으로 누르는 큰 버튼.
///
/// 손떨림이 있어도 누를 수 있게 최소 높이를 120으로 둡니다.
class BigButton extends StatelessWidget {
  const BigButton({
    required this.label,
    required this.onPressed,
    this.icon,
    this.color = ElderColors.action,
    this.minHeight = 120,
    super.key,
  });

  final String label;

  /// null이면 눌리지 않습니다 (연결이 끊겼을 때).
  final VoidCallback? onPressed;
  final IconData? icon;
  final Color color;
  final double minHeight;

  @override
  Widget build(BuildContext context) {
    final icon = this.icon;
    return FilledButton(
      onPressed: onPressed,
      style: FilledButton.styleFrom(
        backgroundColor: color,
        foregroundColor: Colors.white,
        minimumSize: Size(double.infinity, minHeight),
        padding: const EdgeInsets.symmetric(horizontal: 40),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(28)),
        textStyle: const TextStyle(fontSize: 44, fontWeight: FontWeight.w700),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 56),
            const SizedBox(width: 20),
          ],
          Text(label),
        ],
      ),
    );
  }
}
