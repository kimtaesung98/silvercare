import 'package:api_client/api_client.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers.dart';
import 'widgets.dart';

/// 말동무 설정: 켜고 끄기, 안부 전화 시각, 취침 시간, 하루 토큰 한도.
class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({required this.elderId, super.key});

  final String elderId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final s = ref.watch(scheduleProvider(elderId));
    return Scaffold(
      appBar: AppBar(title: const Text('말동무 설정')),
      body: switch (s) {
        AsyncData(:final value) => _Form(elderId: elderId, saved: value),
        AsyncError(:final error) => ErrorRetry(
          error: error,
          onRetry: () => ref.invalidate(scheduleProvider(elderId)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}

class _Form extends ConsumerStatefulWidget {
  const _Form({required this.elderId, required this.saved});

  final String elderId;
  final CompanionSchedule saved;

  @override
  ConsumerState<_Form> createState() => _FormState();
}

class _FormState extends ConsumerState<_Form> {
  late bool _enabled = widget.saved.enabled;
  late List<String> _checkIns = [...widget.saved.checkInTimes];
  late String? _bedStart = widget.saved.bedtimeStart;
  late String? _bedEnd = widget.saved.bedtimeEnd;
  late int? _limit = widget.saved.dailyTokenLimit;
  bool _busy = false;

  /// `HH:MM`을 시계 선택에 쓸 값으로.
  TimeOfDay _time(String hhmm) {
    final p = hhmm.split(':');
    return TimeOfDay(hour: int.parse(p[0]), minute: int.parse(p[1]));
  }

  String _hhmm(TimeOfDay t) =>
      '${t.hour.toString().padLeft(2, '0')}:${t.minute.toString().padLeft(2, '0')}';

  Future<String?> _pick(String? initial) async {
    final t = await showTimePicker(
      context: context,
      initialTime: initial == null
          ? const TimeOfDay(hour: 10, minute: 0)
          : _time(initial),
    );
    return t == null ? null : _hhmm(t);
  }

  Future<void> _addCheckIn() async {
    final v = await _pick(null);
    if (v == null || _checkIns.contains(v)) return;
    setState(() => _checkIns = [..._checkIns, v]..sort());
  }

  Future<void> _editBedtime() async {
    final start = await _pick(_bedStart ?? '21:00');
    if (start == null || !mounted) return;
    final end = await _pick(_bedEnd ?? '07:00');
    if (end == null) return;
    setState(() {
      _bedStart = start;
      _bedEnd = end;
    });
  }

  Future<void> _save() async {
    setState(() => _busy = true);
    try {
      await ref
          .read(backendProvider)
          .saveSchedule(
            widget.elderId,
            CompanionSchedule(
              enabled: _enabled,
              checkInTimes: _checkIns,
              bedtimeStart: _bedStart,
              bedtimeEnd: _bedEnd,
              dailyTokenLimit: _limit,
              timeZone: widget.saved.timeZone,
            ),
          );
      ref.invalidate(scheduleProvider(widget.elderId));
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(const SnackBar(content: Text('저장했어요')));
      }
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
  Widget build(BuildContext context) => ListView(
    padding: const EdgeInsets.all(16),
    children: [
      SwitchListTile(
        value: _enabled,
        onChanged: (v) => setState(() => _enabled = v),
        title: const Text('말동무 켜기'),
        subtitle: const Text('노치원에서 돌아오신 뒤에도 어르신과 이야기를 나눠요'),
      ),
      const Divider(),
      ListTile(
        title: const Text('안부 전화 시각'),
        subtitle: Text(
          _checkIns.isEmpty
              ? '정해진 시각 없음 (어르신이 먼저 말을 거실 때만 대화해요)'
              : _checkIns.join(', '),
        ),
        trailing: IconButton(
          tooltip: '시각 더하기',
          icon: const Icon(Icons.add),
          onPressed: _enabled ? _addCheckIn : null,
        ),
      ),
      if (_checkIns.isNotEmpty)
        Wrap(
          spacing: 8,
          children: [
            for (final t in _checkIns)
              InputChip(
                label: Text(t),
                deleteIcon: const Icon(Icons.close, size: 18),
                onDeleted: () =>
                    setState(() => _checkIns = [..._checkIns]..remove(t)),
              ),
          ],
        ),
      const Divider(),
      ListTile(
        title: const Text('취침 시간'),
        subtitle: Text(
          _bedStart == null || _bedEnd == null
              ? '정해진 시간 없음'
              : '$_bedStart ~ $_bedEnd 에는 말을 걸지 않아요',
        ),
        trailing: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (_bedStart != null)
              IconButton(
                tooltip: '지우기',
                icon: const Icon(Icons.clear),
                onPressed: () => setState(() {
                  _bedStart = null;
                  _bedEnd = null;
                }),
              ),
            IconButton(
              tooltip: '고치기',
              icon: const Icon(Icons.edit_outlined),
              onPressed: _editBedtime,
            ),
          ],
        ),
      ),
      const Divider(),
      ListTile(title: const Text('하루 대화량'), subtitle: Text(_limitLabel())),
      Slider(
        value: (_limit ?? 60000).clamp(10000, 150000).toDouble(),
        min: 10000,
        max: 150000,
        divisions: 14,
        label: _limitLabel(),
        onChanged: (v) => setState(() => _limit = v.round()),
      ),
      if (_limit != null)
        Align(
          alignment: Alignment.centerRight,
          child: TextButton(
            onPressed: () => setState(() => _limit = null),
            child: const Text('센터 기본값으로'),
          ),
        ),
      const SizedBox(height: 24),
      FilledButton(
        onPressed: _busy ? null : _save,
        child: Text(_busy ? '저장 중…' : '저장하기'),
      ),
    ],
  );

  /// 토큰 수는 보호자에게 뜻이 없으니 대화 길이로 바꿔 씁니다.
  String _limitLabel() {
    final l = _limit;
    if (l == null) return '센터 기본값';
    // 한 번 주고받는 데 대략 600토큰이 듭니다.
    return '하루 약 ${(l / 600).round()}번 주고받기';
  }
}
