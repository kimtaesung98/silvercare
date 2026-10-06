import 'package:api_client/api_client.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../backend.dart';
import '../labels.dart';

/// 확인하지 않은 위급 알림 한 건. 누르면 자세히 봅니다.
class EscalationTile extends StatelessWidget {
  const EscalationTile({required this.escalation, super.key});

  final Escalation escalation;

  @override
  Widget build(BuildContext context) {
    final e = escalation;
    final what = e.utteranceText != null
        ? '“${e.utteranceText}”'
        : (e.reason ?? '');
    return Card(
      color: Colors.red.shade700,
      child: ListTile(
        leading: const Icon(Icons.warning_amber_rounded, color: Colors.white),
        title: Text(
          '${e.elder.name} 어르신 · ${triggerLabel(e.triggerType)}',
          style: const TextStyle(
            color: Colors.white,
            fontWeight: FontWeight.w700,
          ),
        ),
        subtitle: Text(
          '${clock(e.createdAt)} $what',
          style: const TextStyle(color: Colors.white),
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
        ),
        trailing: const Text('확인하기', style: TextStyle(color: Colors.white)),
        onTap: () => context.push('/escalations/${e.id}'),
      ),
    );
  }
}

/// 하루 소식 한 건.
class DigestTile extends StatelessWidget {
  const DigestTile({required this.digest, this.now, super.key});

  final DailyDigest digest;

  /// 날짜를 `오늘`, `어제`로 쓸 기준. 기본은 지금입니다.
  final DateTime? now;

  @override
  Widget build(BuildContext context) {
    final d = digest;
    final flag = d.emotionFlag;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text(
                  dayOrToday(d.date, now ?? DateTime.now()),
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const Spacer(),
                Text(
                  d.sessionCount == 0 ? '대화 없음' : '대화 ${d.sessionCount}번',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(d.summaryText, style: const TextStyle(fontSize: 16)),
            if (flag != null) ...[
              const SizedBox(height: 8),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Icon(
                    d.escalationCount > 0
                        ? Icons.warning_amber_rounded
                        : Icons.info_outline,
                    size: 20,
                    color: d.escalationCount > 0
                        ? Colors.red.shade700
                        : Theme.of(context).colorScheme.primary,
                  ),
                  const SizedBox(width: 6),
                  Expanded(child: Text(flag)),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// 불러오기 실패와 다시 시도 버튼.
class ErrorRetry extends StatelessWidget {
  const ErrorRetry({required this.error, required this.onRetry, super.key});

  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 32),
    child: Column(
      children: [
        Text(describe(error), textAlign: TextAlign.center),
        const SizedBox(height: 8),
        OutlinedButton(onPressed: onRetry, child: const Text('다시 불러오기')),
      ],
    ),
  );
}

/// 오류를 보호자가 읽을 말로 바꿉니다.
String describe(Object error) => switch (error) {
  SignedOutException() => '$error',
  ApiException(:final code) => '서버 오류가 났어요 ($code)',
  _ => '서버에 연결하지 못했어요',
};
