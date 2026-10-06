import 'package:flutter/material.dart';

import 'elder_theme.dart';

/// 화면에 띄우는 말 한 줄.
class CaptionLine {
  const CaptionLine({required this.text, required this.isElder});

  final String text;

  /// 어르신이 한 말인지 (아니면 AI).
  final bool isElder;
}

/// 방금 나눈 말을 큰 글씨로 보여줍니다.
///
/// 귀가 어두운 어르신이 소리를 못 들었을 때 글자로 따라갈 수 있게 둡니다
/// (architecture.md 6.1절). 새 줄이 아래에 옵니다.
class CaptionBoard extends StatelessWidget {
  const CaptionBoard({required this.lines, super.key});

  final List<CaptionLine> lines;

  @override
  Widget build(BuildContext context) {
    if (lines.isEmpty) return const SizedBox.shrink();
    return Column(
      mainAxisAlignment: MainAxisAlignment.end,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (final line
            in lines.length > 3 ? lines.sublist(lines.length - 3) : lines)
          Padding(
            padding: const EdgeInsets.only(bottom: 16),
            child: Align(
              alignment: line.isElder
                  ? Alignment.centerRight
                  : Alignment.centerLeft,
              child: Container(
                constraints: const BoxConstraints(maxWidth: 760),
                padding: const EdgeInsets.symmetric(
                  horizontal: 28,
                  vertical: 20,
                ),
                decoration: BoxDecoration(
                  color: line.isElder
                      ? ElderColors.elderBubble
                      : ElderColors.aiBubble,
                  borderRadius: BorderRadius.circular(24),
                ),
                child: Text(
                  line.text,
                  style: Theme.of(context).textTheme.bodyLarge,
                ),
              ),
            ),
          ),
      ],
    );
  }
}
