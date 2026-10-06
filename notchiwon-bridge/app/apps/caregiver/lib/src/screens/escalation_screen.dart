import 'package:api_client/api_client.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../labels.dart';
import '../providers.dart';
import 'widgets.dart';

/// 위급 알림 한 건. 확인하면 미확인 목록과 재알림에서 빠집니다.
class EscalationScreen extends ConsumerStatefulWidget {
  const EscalationScreen({required this.escalationId, super.key});

  final String escalationId;

  @override
  ConsumerState<EscalationScreen> createState() => _EscalationScreenState();
}

class _EscalationScreenState extends ConsumerState<EscalationScreen> {
  bool _busy = false;

  Future<void> _ack() async {
    setState(() => _busy = true);
    try {
      await ref.read(backendProvider).ackEscalation(widget.escalationId);
      ref.invalidate(escalationProvider(widget.escalationId));
      ref.invalidate(openEscalationsProvider);
    } on Object catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(describe(e))));
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final e = ref.watch(escalationProvider(widget.escalationId));
    return Scaffold(
      appBar: AppBar(title: const Text('위급 알림')),
      body: switch (e) {
        AsyncData(:final value) => _body(context, value),
        AsyncError(:final error) => ErrorRetry(
          error: error,
          onRetry: () =>
              ref.invalidate(escalationProvider(widget.escalationId)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }

  Widget _body(BuildContext context, Escalation e) {
    final text = Theme.of(context).textTheme;
    final acked = e.acknowledgedAt;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('${e.elder.name} 어르신', style: text.headlineSmall),
        const SizedBox(height: 4),
        Text(
          '${triggerLabel(e.triggerType)} · ${clock(e.createdAt)}',
          style: text.titleMedium?.copyWith(color: Colors.red.shade700),
        ),
        const SizedBox(height: 16),
        if (e.utteranceText != null)
          Text('“${e.utteranceText}”', style: text.headlineSmall),
        if (e.reason != null) ...[
          const SizedBox(height: 8),
          Text(e.reason!, style: text.bodyLarge),
        ],
        const SizedBox(height: 8),
        const Text('어르신 태블릿은 안심 문장으로 답했어요. 상태를 직접 확인해 주세요.'),
        const SizedBox(height: 24),
        if (acked == null)
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red.shade700),
            onPressed: _busy ? null : _ack,
            child: const Text('확인했어요'),
          )
        else
          Text('${clock(acked)}에 확인함', style: text.titleMedium),
        if (e.visitId != null) ...[
          const SizedBox(height: 8),
          OutlinedButton(
            onPressed: () => context.push('/visits/${e.visitId}'),
            child: const Text('방문 화면 열기'),
          ),
        ],
      ],
    );
  }
}
