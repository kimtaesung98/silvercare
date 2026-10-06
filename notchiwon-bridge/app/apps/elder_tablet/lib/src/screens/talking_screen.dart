import 'package:flutter/material.dart';
import 'package:ui/ui.dart';

/// 대화 화면: 방금 나눈 말과, 지금 듣고 있는지 말하고 있는지를 보여줍니다.
class TalkingScreen extends StatelessWidget {
  const TalkingScreen({
    required this.captions,
    required this.speaking,
    required this.onEnd,
    this.etaMinutes,
    super.key,
  });

  final List<CaptionLine> captions;

  /// AI가 말하는 중인지 (아니면 듣는 중).
  final bool speaking;

  /// 그만하기 버튼.
  final VoidCallback onEnd;

  /// 조무사 도착까지 남은 분 (픽업 브릿지에서만).
  final int? etaMinutes;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final eta = etaMinutes;
    return Padding(
      padding: const EdgeInsets.fromLTRB(40, 32, 40, 40),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Icon(
                speaking ? Icons.volume_up : Icons.mic,
                size: 48,
                color: ElderColors.action,
              ),
              const SizedBox(width: 16),
              Text(speaking ? '이야기하고 있어요' : '듣고 있어요', style: text.titleLarge),
              const Spacer(),
              if (eta != null) Text('$eta분 뒤 도착', style: text.bodyMedium),
            ],
          ),
          const SizedBox(height: 24),
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
    );
  }
}
