import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:ui/ui.dart';

import 'mic_meter.dart';

/// 대기 화면: 세션이 없을 때. 말동무 버튼과, 마이크가 듣고 있는지 보여주는
/// 소리 막대를 둡니다. 말을 걸어도 말동무가 시작됩니다.
class IdleScreen extends StatelessWidget {
  const IdleScreen({
    required this.elderName,
    required this.onCompanion,
    required this.micLevel,
    this.micOn,
    this.connected = true,
    super.key,
  });

  /// 어르신 호칭 ("김영자 어르신").
  final String elderName;

  /// 말동무 버튼. 연결이 끊겼으면 null입니다.
  final VoidCallback? onCompanion;
  final bool connected;

  /// 소리 크기 (0~1).
  final ValueListenable<double> micLevel;

  /// 마이크가 열렸는지 (여는 중이면 null).
  final bool? micOn;

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
            '심심하시면 말을 걸거나 눌러 주세요',
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
          MicMeter(level: micLevel, on: micOn),
          const SizedBox(height: 24),
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
