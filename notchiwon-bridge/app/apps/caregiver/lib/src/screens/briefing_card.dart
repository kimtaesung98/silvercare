import 'package:api_client/api_client.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../labels.dart';
import '../providers.dart';
import 'widgets.dart';

/// 도착 브리핑. 픽업 대기 대화 요약과 자주 나온 이야기, 살펴볼 점을 보여줍니다.
class BriefingCard extends ConsumerWidget {
  const BriefingCard({required this.sessionId, super.key});

  final String sessionId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final b = ref.watch(briefingProvider(sessionId));
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: switch (b) {
          AsyncData(:final value?) => _Briefing(briefing: value),
          AsyncError(:final error) => ErrorRetry(
            error: error,
            onRetry: () => ref.invalidate(briefingProvider(sessionId)),
          ),
          _ => const Row(
            children: [
              SizedBox.square(
                dimension: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
              SizedBox(width: 12),
              Text('브리핑을 만드는 중이에요…'),
            ],
          ),
        },
      ),
    );
  }
}

class _Briefing extends StatelessWidget {
  const _Briefing({required this.briefing});

  final Briefing briefing;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final flag = briefing.emotionFlag;
    final tag = briefing.overallEmotionTag;
    final escalations = briefing.escalationCount ?? 0;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('도착 브리핑', style: text.titleLarge),
        const SizedBox(height: 8),
        if (flag != null || escalations > 0)
          Container(
            width: double.infinity,
            margin: const EdgeInsets.only(bottom: 8),
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.orange.shade100,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text(
              [
                ?flag,
                if (escalations > 0) '대화 중 위급 감지 $escalations건',
              ].join('\n'),
              style: const TextStyle(fontWeight: FontWeight.w600),
            ),
          ),
        Text(briefing.summaryText, style: text.bodyLarge),
        if (briefing.topKeywords.isNotEmpty) ...[
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 4,
            children: [
              for (final k in briefing.topKeywords) Chip(label: Text(k)),
            ],
          ),
        ],
        if (tag != null) ...[
          const SizedBox(height: 8),
          Text('기분: ${emotionLabel(tag)}', style: text.bodyMedium),
        ],
      ],
    );
  }
}
