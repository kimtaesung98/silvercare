import 'package:api_client/api_client.dart';
import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:latlong2/latlong.dart';
import 'package:url_launcher/url_launcher.dart';

import '../labels.dart';
import '../map_tiles.dart';
import '../providers.dart';
import '../trip.dart';
import 'briefing_card.dart';
import 'widgets.dart';

/// 방문 한 건: 지도와 남은 시간, 출발·도착 버튼, 도착 뒤 브리핑 카드.
class VisitScreen extends ConsumerWidget {
  const VisitScreen({required this.visitId, super.key});

  final String visitId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final visit = ref.watch(visitProvider(visitId));
    return Scaffold(
      appBar: AppBar(
        title: Text(
          visit.hasValue ? '${visit.requireValue.elder.name} 어르신' : '방문',
        ),
      ),
      body: switch (visit) {
        AsyncData(:final value) => _VisitBody(visit: value),
        AsyncError(:final error) => ErrorRetry(
          error: error,
          onRetry: () => ref.invalidate(visitProvider(visitId)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}

class _VisitBody extends ConsumerStatefulWidget {
  const _VisitBody({required this.visit});

  final Visit visit;

  @override
  ConsumerState<_VisitBody> createState() => _VisitBodyState();
}

class _VisitBodyState extends ConsumerState<_VisitBody> {
  bool _busy = false;

  Visit get visit => widget.visit;

  void _say(String text) =>
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(text)));

  Future<void> _depart() async {
    setState(() => _busy = true);
    final access = await ref.read(tripProvider.notifier).start(visit.id);
    if (!mounted) return;
    setState(() => _busy = false);
    switch (access) {
      case LocationAccess.granted:
        break;
      case LocationAccess.denied:
        _say('위치 권한을 허용해야 도착 시간을 어르신께 알려드릴 수 있어요');
      case LocationAccess.serviceOff:
        _say('휴대폰의 위치(GPS)를 켜 주세요');
    }
  }

  Future<void> _arrive() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (c) => AlertDialog(
        title: const Text('도착했나요?'),
        content: Text('${visit.elder.name} 어르신 태블릿의 대화를 마치고 브리핑을 준비해요.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(c, false),
            child: const Text('아니요'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(c, true),
            child: const Text('도착했어요'),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    setState(() => _busy = true);
    try {
      await ref.read(backendProvider).arrive(visit.id);
      if (ref.read(tripProvider)?.visitId == visit.id) {
        ref.read(tripProvider.notifier).stop();
      }
    } on Object catch (e) {
      _say(describe(e));
    } finally {
      ref.invalidate(visitProvider(visit.id));
      ref.invalidate(todayVisitsProvider);
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _navigate(Place to) async {
    final name = Uri.encodeComponent('${visit.elder.name} 어르신 댁');
    final app = Uri.parse(
      'kakaomap://route?ep=${to.latitude},${to.longitude}&by=CAR',
    );
    final web = Uri.parse(
      'https://map.kakao.com/link/to/$name,${to.latitude},${to.longitude}',
    );
    if (!await launchUrl(app, mode: LaunchMode.externalApplication)) {
      await launchUrl(web, mode: LaunchMode.externalApplication);
    }
  }

  @override
  Widget build(BuildContext context) {
    final trip = ref.watch(tripProvider);
    final moving = trip?.visitId == visit.id ? trip : null;
    final alerts = (ref.watch(openEscalationsProvider).value ?? const []).where(
      (e) => e.visitId == visit.id,
    );
    final to = visit.destination;
    final eta =
        moving?.etaMinutes ??
        (visit.etaCurrent != null && isOpen(visit.status)
            ? minutesUntil(visit.etaCurrent!, DateTime.now())
            : null);
    final text = Theme.of(context).textTheme;

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        for (final e in alerts) EscalationTile(escalation: e),
        if (to != null)
          _VisitMap(destination: to, here: moving?.lastFix)
        else
          const Card(
            child: Padding(
              padding: EdgeInsets.all(16),
              child: Text('어르신 댁 위치가 등록되지 않았어요. 센터에 알려 주세요.'),
            ),
          ),
        const SizedBox(height: 12),
        if (to?.address != null) Text(to!.address!, style: text.bodyLarge),
        Text(
          '${clock(visit.scheduledTime)} 예정 · ${visitStatusLabel(visit.status)}',
          style: text.bodyMedium,
        ),
        if (eta != null) ...[
          const SizedBox(height: 8),
          Text('도착까지 약 $eta분', style: text.headlineMedium),
        ],
        if (moving != null) ...[
          const SizedBox(height: 4),
          Text(
            moving.error ??
                (moving.sessionStarted
                    ? '어르신 태블릿에서 대화를 시작했어요'
                    : '10초마다 위치를 보내고 있어요'),
            style: TextStyle(
              color: moving.error != null
                  ? Theme.of(context).colorScheme.error
                  : null,
            ),
          ),
        ],
        const SizedBox(height: 16),
        if (isOpen(visit.status)) ...[
          if (moving == null)
            FilledButton.icon(
              onPressed: _busy ? null : _depart,
              icon: const Icon(Icons.directions_car),
              label: const Text('출발하기'),
            )
          else
            OutlinedButton(
              onPressed: () => ref.read(tripProvider.notifier).stop(),
              child: const Text('위치 보내기 멈추기'),
            ),
          if (to != null) ...[
            const SizedBox(height: 8),
            OutlinedButton.icon(
              onPressed: () => _navigate(to),
              icon: const Icon(Icons.navigation_outlined),
              label: const Text('카카오맵으로 길안내'),
            ),
          ],
          const SizedBox(height: 8),
          FilledButton.tonal(
            onPressed: _busy ? null : _arrive,
            child: const Text('도착했어요'),
          ),
        ],
        if (visit.status == VisitStatus.COMPLETED) ...[
          if (visit.sessionId != null)
            BriefingCard(sessionId: visit.sessionId!)
          else
            const Text('이번 방문에는 어르신과 나눈 대화가 없어요.'),
        ],
      ],
    );
  }
}

class _VisitMap extends ConsumerWidget {
  const _VisitMap({required this.destination, this.here});

  final Place destination;
  final Fix? here;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final tiles = ref.watch(mapTilesProvider);
    final home = LatLng(destination.latitude, destination.longitude);
    final me = here == null ? null : LatLng(here!.latitude, here!.longitude);
    return SizedBox(
      height: 260,
      child: ClipRRect(
        borderRadius: BorderRadius.circular(12),
        child: FlutterMap(
          key: ValueKey(me == null),
          options: MapOptions(
            initialCenter: home,
            initialZoom: 15,
            initialCameraFit: me == null
                ? null
                : CameraFit.coordinates(
                    coordinates: [home, me],
                    padding: const EdgeInsets.all(48),
                    maxZoom: 16,
                  ),
          ),
          children: [
            ?tiles,
            MarkerLayer(
              markers: [
                Marker(
                  point: home,
                  width: 40,
                  height: 40,
                  child: const Icon(Icons.home, size: 36, color: Colors.red),
                ),
                if (me != null)
                  Marker(
                    point: me,
                    width: 32,
                    height: 32,
                    child: const Icon(
                      Icons.directions_car,
                      size: 28,
                      color: Colors.blue,
                    ),
                  ),
              ],
            ),
            if (tiles != null)
              const SimpleAttributionWidget(
                source: Text('OpenStreetMap contributors'),
              ),
          ],
        ),
      ),
    );
  }
}
