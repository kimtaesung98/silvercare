import 'package:flutter/material.dart';
import 'package:ui/ui.dart';

/// 도착 임박 화면: 조무사가 거의 다 왔을 때. 대화는 그대로 이어가되,
/// "곧 와요"를 가장 크게 보여줘서 어르신이 현관으로 나설 수 있게 합니다.
class ArrivingScreen extends StatelessWidget {
  const ArrivingScreen({
    required this.minutes,
    required this.captions,
    required this.onEnd,
    this.caregiverName,
    super.key,
  });

  /// 남은 분.
  final int minutes;
  final List<CaptionLine> captions;
  final VoidCallback onEnd;

  /// 조무사 이름 ("박선생님").
  final String? caregiverName;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final who = caregiverName ?? '선생님';
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Container(
          color: ElderColors.arriving,
          padding: const EdgeInsets.symmetric(vertical: 40, horizontal: 32),
          child: Column(
            children: [
              Icon(Icons.directions_car, size: 96, color: Colors.white),
              const SizedBox(height: 16),
              Text(
                minutes <= 0 ? '$who이 도착했어요' : '$who이 $minutes분 뒤에 와요',
                textAlign: TextAlign.center,
                style: text.headlineLarge?.copyWith(color: Colors.white),
              ),
              const SizedBox(height: 12),
              Text(
                '천천히 준비하세요',
                textAlign: TextAlign.center,
                style: text.bodyLarge?.copyWith(color: Colors.white),
              ),
            ],
          ),
        ),
        Expanded(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(40, 24, 40, 40),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Expanded(child: CaptionBoard(lines: captions)),
                const SizedBox(height: 24),
                BigButton(
                  label: '그만하기',
                  icon: Icons.close,
                  color: ElderColors.quiet,
                  minHeight: 110,
                  onPressed: onEnd,
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}
