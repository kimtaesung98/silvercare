import 'dart:async';

import 'package:api_client/api_client.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../providers.dart';
import 'widgets.dart';

/// 돌보는 어르신 목록. 확인하지 않은 위급 알림이 있으면 맨 위에 빨갛게 보여줍니다.
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
    ref.invalidate(eldersProvider);
    await ref.read(eldersProvider.future).catchError((_) => <ElderSummary>[]);
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authProvider);
    final name = auth?.guardianName ?? '';
    // 서버에서 다시 읽기 전에는 로그인할 때 받은 목록을 보여줍니다.
    final elders = ref.watch(eldersProvider);
    final shown = elders.value ?? auth?.elders ?? const <ElderSummary>[];
    final alerts = ref.watch(openEscalationsProvider).value ?? const [];

    return Scaffold(
      appBar: AppBar(
        title: Text('$name님'),
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
            if (shown.isEmpty)
              switch (elders) {
                AsyncError(:final error) => ErrorRetry(
                  error: error,
                  onRetry: _reload,
                ),
                AsyncData() => const Padding(
                  padding: EdgeInsets.symmetric(vertical: 48),
                  child: Center(child: Text('돌보시는 어르신이 아직 등록되지 않았어요')),
                ),
                _ => const Padding(
                  padding: EdgeInsets.all(48),
                  child: Center(child: CircularProgressIndicator()),
                ),
              }
            else
              for (final e in shown) _ElderTile(elder: e),
          ],
        ),
      ),
    );
  }
}

class _ElderTile extends StatelessWidget {
  const _ElderTile({required this.elder});

  final ElderSummary elder;

  @override
  Widget build(BuildContext context) => Card(
    child: ListTile(
      leading: const Icon(Icons.elderly),
      title: Text(
        '${elder.name} 어르신',
        style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
      ),
      subtitle: const Text('하루 소식 보기'),
      trailing: const Icon(Icons.chevron_right),
      onTap: () => context.push('/elders/${elder.id}'),
    ),
  );
}
