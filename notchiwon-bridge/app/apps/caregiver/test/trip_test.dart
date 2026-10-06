import 'package:api_client/api_client.dart';
import 'package:caregiver/src/backend.dart';
import 'package:caregiver/src/providers.dart';
import 'package:caregiver/src/trip.dart';
import 'package:fake_async/fake_async.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fakes.dart';

void main() {
  late FakeBackend backend;
  late FakePositions positions;
  late ProviderContainer c;

  setUp(() {
    backend = FakeBackend()..visits['v1'] = visit();
    positions = FakePositions();
    c = ProviderContainer(
      overrides: overrides(backend: backend, positions: positions),
    );
  });
  tearDown(() => c.dispose());

  test('첫 위치는 바로, 그다음은 10초마다 가장 최근 위치를 보낸다', () {
    fakeAsync((async) {
      c.read(tripProvider.notifier).start('v1');
      async.flushMicrotasks();
      expect(c.read(tripProvider)?.visitId, 'v1');
      expect(backend.sent, isEmpty);

      positions.move(37.50, 127.00, now);
      async.flushMicrotasks();
      expect(backend.sent, hasLength(1));
      expect(c.read(tripProvider)?.etaMinutes, 12);

      positions.move(37.51, 127.00, now.add(const Duration(seconds: 3)));
      positions.move(37.52, 127.00, now.add(const Duration(seconds: 6)));
      async.elapse(const Duration(seconds: 10));
      expect(backend.sent, hasLength(2));
      expect(backend.sent.last.$2.latitude, 37.52);
      expect(
        backend.sent.last.$2.recordedAt,
        now.add(const Duration(seconds: 6)),
      );

      // 차가 서 있어도 10초마다 보내 ETA가 갱신됩니다.
      async.elapse(const Duration(seconds: 20));
      expect(backend.sent, hasLength(4));
    });
  });

  test('세션이 시작되면 표시하고, 방문이 끝나면 멈춘다', () {
    fakeAsync((async) {
      backend.onLocation = (id) => LocationUpdateResult(
        visitId: id,
        status: VisitStatus.SESSION_ACTIVE,
        etaMinutes: 9,
        sessionStarted: true,
        sessionId: 's1',
      );
      c.read(tripProvider.notifier).start('v1');
      async.flushMicrotasks();
      positions.move(37.5, 127, now);
      async.flushMicrotasks();
      expect(c.read(tripProvider)?.sessionStarted, isTrue);

      backend.locationError = const VisitClosedException();
      async.elapse(const Duration(seconds: 10));
      expect(c.read(tripProvider), isNull);
      async.elapse(const Duration(seconds: 30));
      expect(backend.sent, hasLength(2));
    });
  });

  test('보내기가 실패하면 알리고 계속 시도한다', () {
    fakeAsync((async) {
      c.read(tripProvider.notifier).start('v1');
      async.flushMicrotasks();
      backend.locationError = Exception('offline');
      positions.move(37.5, 127, now);
      async.flushMicrotasks();
      expect(c.read(tripProvider)?.error, isNotNull);

      backend.locationError = null;
      async.elapse(const Duration(seconds: 10));
      expect(c.read(tripProvider)?.error, isNull);
      expect(c.read(tripProvider)?.etaMinutes, 12);
    });
  });

  test('위치 권한이 없으면 출발하지 않는다', () async {
    positions.access = LocationAccess.denied;
    expect(
      await c.read(tripProvider.notifier).start('v1'),
      LocationAccess.denied,
    );
    expect(c.read(tripProvider), isNull);
  });

  test('로그아웃하면 이동도 멈춘다', () async {
    c.listen(authProvider, (_, _) {});
    await c.read(tripProvider.notifier).start('v1');
    await c.read(authProvider.notifier).signOut();
    expect(c.read(tripProvider), isNull);
    expect(c.read(authProvider), isNull);
  });
}
