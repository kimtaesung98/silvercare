import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:ui/ui.dart';

/// 마이크가 듣고 있는지 보여주는 줄: 마이크 그림과 소리 크기 막대 다섯 개.
/// 어르신이 말하면 막대가 올라가서, 태블릿이 듣고 있다는 것을 알 수 있습니다.
class MicMeter extends StatelessWidget {
  const MicMeter({required this.level, required this.on, super.key});

  /// 소리 크기 (0~1).
  final ValueListenable<double> level;

  /// 마이크가 열렸는지 (여는 중이면 null).
  final bool? on;

  static const _bars = 5;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    if (on == false) {
      return Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.mic_off, size: 40, color: ElderColors.quiet),
          const SizedBox(width: 12),
          Text('마이크를 쓸 수 없어요', style: text.bodyMedium),
        ],
      );
    }
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Icon(Icons.mic, size: 40, color: ElderColors.action),
        const SizedBox(width: 12),
        ValueListenableBuilder<double>(
          valueListenable: level,
          builder: (context, value, _) => Row(
            key: const Key('mic-bars'),
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              for (var i = 0; i < _bars; i++)
                Container(
                  width: 10,
                  height: 12.0 + 6 * i,
                  margin: const EdgeInsets.symmetric(horizontal: 3),
                  decoration: BoxDecoration(
                    color: value * _bars > i
                        ? ElderColors.action
                        : ElderColors.action.withValues(alpha: 0.15),
                    borderRadius: BorderRadius.circular(3),
                  ),
                ),
            ],
          ),
        ),
      ],
    );
  }
}
