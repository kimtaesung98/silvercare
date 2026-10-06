import 'package:api_client/api_client.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../labels.dart';
import '../providers.dart';
import 'widgets.dart';

/// 어르신 한 분의 하루 소식 목록. 위에서 말동무 설정으로 갈 수 있습니다.
class ElderScreen extends ConsumerWidget {
  const ElderScreen({required this.elderId, super.key});

  final String elderId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final name = ref
        .watch(authProvider)
        ?.elders
        .where((e) => e.id == elderId)
        .map((e) => e.name)
        .firstOrNull;
    final digests = ref.watch(digestsProvider(elderId));
    final schedule = ref.watch(scheduleProvider(elderId)).value;

    return Scaffold(
      appBar: AppBar(
        title: Text(name == null ? '하루 소식' : '$name 어르신'),
        actions: [
          IconButton(
            tooltip: '말동무 설정',
            icon: const Icon(Icons.settings_outlined),
            onPressed: () => context.push('/elders/$elderId/settings'),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async {
          ref.invalidate(digestsProvider(elderId));
          await ref
              .read(digestsProvider(elderId).future)
              .catchError((_) => <DailyDigest>[]);
        },
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            if (schedule != null)
              Card(
                child: ListTile(
                  leading: Icon(
                    schedule.enabled
                        ? Icons.record_voice_over_outlined
                        : Icons.voice_over_off_outlined,
                  ),
                  title: Text(schedule.enabled ? '말동무 켜짐' : '말동무 꺼짐'),
                  subtitle: Text(
                    schedule.enabled
                        ? '안부 전화 ${checkInLabel(schedule.checkInTimes)}'
                        : '어르신이 말을 거셔도 대화하지 않아요',
                  ),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: () => context.push('/elders/$elderId/settings'),
                ),
              ),
            ...switch (digests) {
              AsyncData(:final value) when value.isEmpty => [
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 48),
                  child: Center(child: Text('아직 하루 소식이 없어요')),
                ),
              ],
              AsyncData(:final value) => [
                for (final d in value) DigestTile(digest: d),
              ],
              AsyncError(:final error) => [
                ErrorRetry(
                  error: error,
                  onRetry: () => ref.invalidate(digestsProvider(elderId)),
                ),
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
