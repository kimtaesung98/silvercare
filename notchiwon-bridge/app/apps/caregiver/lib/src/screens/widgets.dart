import 'package:api_client/api_client.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../backend.dart';
import '../labels.dart';
import '../providers.dart';
import '../trip.dart';

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

/// 이동 중인 방문을 알려주는 띠.
class TripBanner extends StatelessWidget {
  const TripBanner({required this.trip, super.key});

  final Trip trip;

  @override
  Widget build(BuildContext context) {
    final eta = trip.etaMinutes;
    final scheme = Theme.of(context).colorScheme;
    return Card(
      color: scheme.primaryContainer,
      child: ListTile(
        leading: const Icon(Icons.my_location),
        title: Text(eta == null ? '위치를 보내는 중이에요' : '도착까지 약 $eta분'),
        subtitle: Text(
          trip.error ??
              (trip.sessionStarted ? '어르신 태블릿에서 대화를 시작했어요' : '10초마다 위치를 보내요'),
        ),
        onTap: () => context.push('/visits/${trip.visitId}'),
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

/// 오류를 조무사가 읽을 말로 바꿉니다.
String describe(Object error) => switch (error) {
  SignedOutException() ||
  VisitClosedException() ||
  BriefingTimeout() => '$error',
  ApiException(:final code) => '서버 오류가 났어요 ($code)',
  _ => '서버에 연결하지 못했어요',
};
