import 'package:flutter/material.dart';

import 'elder_theme.dart';

/// 화면 위쪽의 안내 띠 (거절 사유, 연결 끊김, 못 알아들었을 때).
class NoticeBar extends StatelessWidget {
  const NoticeBar({required this.text, super.key});

  final String text;

  @override
  Widget build(BuildContext context) => Container(
    width: double.infinity,
    color: ElderColors.notice,
    padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 24),
    child: Text(
      text,
      textAlign: TextAlign.center,
      style: Theme.of(context).textTheme.titleLarge,
    ),
  );
}
