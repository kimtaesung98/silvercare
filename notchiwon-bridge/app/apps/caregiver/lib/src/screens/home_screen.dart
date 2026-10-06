import 'dart:async';

import 'package:api_client/api_client.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../labels.dart';
import '../providers.dart';
import 'widgets.dart';

/// 오늘 방문 목록. 확인하지 않은 위급 알림이 있으면 맨 위에 빨갛게 보여줍니다.
class HomeScreen extends ConsumerStatefulWidget {
  const HomeScreen({super.key});

  @override
  ConsumerState<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends ConsumerState<HomeScreen> {
  Timer? _refresh;

  @override
  void initState() {
    super.initState();
    // 푸시를 못 받는 기기도 30초 안에는 위급 알림을 봅니다.
    _refresh = Timer.periodic(const Duration(seconds: 30), (_) => _reload());
  }

  @override
  void dispose() {
    _refresh?.cancel();
    super.dispose();
  }

  Future<void> _reload() async {
    ref.invalidate(openEscalationsProvider);
    ref.invalidate(todayVisitsProvider);
    await ref.read(todayVisitsProvider.future).catchError((_) => <Visit>[]);
  }

  @override
  Widget build(BuildContext context) {
    final name = ref.watch(authProvider)?.caregiverName ?? '';
    final visits = ref.watch(todayVisitsProvider);
    final alerts = ref.watch(openEscalationsProvider).value ?? const [];
    final trip = ref.watch(tripProvider);

    return Scaffold(
      appBar: AppBar(
        title: Text('$name 선생님의 오늘 방문'),
        actions: [
          PopupMenuButton<void>(
            itemBuilder: (_) => [
              PopupMenuItem(
                onTap: () => ref.read(authProvider.notifier).signOut(),
                child: const Text('로그아웃'),
              ),
            ],
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _reload,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            for (final e in alerts) EscalationTile(escalation: e),
            if (trip != null) TripBanner(trip: trip),
            ...switch (visits) {
              AsyncData(:final value) when value.isEmpty => [
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 48),
                  child: Center(child: Text('오늘 예정된 방문이 없어요')),
                ),
              ],
              AsyncData(:final value) => [
                for (final v in value) _VisitTile(visit: v),
              ],
              AsyncError(:final error) => [
                ErrorRetry(error: error, onRetry: _reload),
              ],
              _ => [
                const Padding(
                  padding: EdgeInsets.all(48),
                  child: Center(child: CircularProgressIndicator()),
                ),
              ],
            },
          ],
        ),
      ),
    );
  }
}

class _VisitTile extends ConsumerWidget {
  const _VisitTile({required this.visit});

  final Visit visit;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final trip = ref.watch(tripProvider);
    final moving = trip?.visitId == visit.id;
    final eta = moving ? trip?.etaMinutes : null;
    final subtitle = [
      '${clock(visit.scheduledTime)} 예정',
      visitStatusLabel(visit.status),
      if (eta != null) '약 $eta분 남음',
    ].join(' · ');
    return Card(
      child: ListTile(
        leading: Icon(
          moving ? Icons.directions_car : Icons.home_outlined,
          color: moving ? Theme.of(context).colorScheme.primary : null,
        ),
        title: Text(
          '${visit.elder.name} 어르신',
          style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
        ),
        subtitle: Text(subtitle),
        trailing: const Icon(Icons.chevron_right),
        onTap: () => context.push('/visits/${visit.id}'),
      ),
    );
  }
}
