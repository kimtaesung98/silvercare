import 'package:flutter/material.dart';
import 'package:ui/ui.dart';

/// 대기 화면: 세션이 없을 때. 말동무 버튼 하나만 둡니다.
class IdleScreen extends StatelessWidget {
  const IdleScreen({
    required this.elderName,
    required this.onCompanion,
    this.connected = true,
    super.key,
  });

  /// 어르신 호칭 ("김영자 어르신").
  final String elderName;

  /// 말동무 버튼. 연결이 끊겼으면 null입니다.
  final VoidCallback? onCompanion;
  final bool connected;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Padding(
      padding: const EdgeInsets.all(56),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(elderName, textAlign: TextAlign.center, style: text.titleLarge),
          const SizedBox(height: 24),
          Text(
            '심심하시면 눌러 주세요',
            textAlign: TextAlign.center,
            style: text.headlineLarge,
          ),
          const SizedBox(height: 72),
          BigButton(
            label: '말동무 하기',
            icon: Icons.favorite,
            minHeight: 180,
            onPressed: connected ? onCompanion : null,
          ),
          const SizedBox(height: 40),
          if (!connected)
            Text(
              '연결을 다시 하고 있어요',
              textAlign: TextAlign.center,
              style: text.bodyMedium,
            ),
        ],
      ),
    );
  }
}
