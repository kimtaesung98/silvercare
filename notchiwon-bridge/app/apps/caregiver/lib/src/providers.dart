import 'dart:async';

import 'package:api_client/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'backend.dart';
import 'credential_store.dart';
import 'push.dart';
import 'trip.dart';

/// `main`에서 실제 구현으로, 테스트에서 가짜로 바꿔 끼웁니다.
final backendProvider = Provider<Backend>(
  (ref) => throw UnimplementedError('backendProvider'),
);
final credentialStoreProvider = Provider<CredentialStore>(
  (ref) => MemoryCredentialStore(),
);

/// 앱을 켤 때 보관소에서 읽은 로그인 정보.
final initialCredentialsProvider = Provider<Credentials?>((ref) => null);
final positionSourceProvider = Provider<PositionSource>(
  (ref) => const GeolocatorSource(),
);
final pushProvider = Provider<Push>((ref) => const NoPush());

/// 위치 전송 간격.
final reportIntervalProvider = Provider<Duration>(
  (ref) => const Duration(seconds: 10),
);

/// 브리핑이 아직 없을 때 다시 물어보는 간격과 횟수.
final briefingPollProvider = Provider<({Duration every, int tries})>(
  (ref) => (every: const Duration(seconds: 3), tries: 40),
);

/// 로그인 상태. null이면 로그인 화면으로 갑니다.
class Auth extends Notifier<Credentials?> {
  @override
  Credentials? build() {
    final c = ref.read(initialCredentialsProvider);
    return c == null || c.expiredAt(DateTime.now()) ? null : c;
  }

  Future<void> login(String loginId, String password) async {
    final c = await ref.read(backendProvider).login(loginId, password);
    await ref.read(credentialStoreProvider).write(c);
    state = c;
  }

  Future<void> signOut() async {
    ref.read(tripProvider.notifier).stop();
    await ref.read(credentialStoreProvider).clear();
    state = null;
  }
}

final authProvider = NotifierProvider<Auth, Credentials?>(Auth.new);

final tripProvider = NotifierProvider<TripController, Trip?>(
  TripController.new,
);

final todayVisitsProvider = FutureProvider.autoDispose<List<Visit>>(
  (ref) => ref.watch(backendProvider).todayVisits(),
);

final visitProvider = FutureProvider.autoDispose.family<Visit, String>(
  (ref, id) => ref.watch(backendProvider).visit(id),
);

final openEscalationsProvider = FutureProvider.autoDispose<List<Escalation>>(
  (ref) => ref.watch(backendProvider).openEscalations(),
);

final escalationProvider = FutureProvider.autoDispose
    .family<Escalation, String>(
      (ref, id) => ref.watch(backendProvider).escalation(id),
    );

/// 브리핑이 나올 때까지 몇 초마다 다시 묻습니다. 만들어지는 동안은 null,
/// 끝내 나오지 않으면 [BriefingTimeout] 오류입니다. 받으면 읽음으로 표시합니다.
final briefingProvider = StreamProvider.autoDispose.family<Briefing?, String>((
  ref,
  sessionId,
) async* {
  final backend = ref.watch(backendProvider);
  final poll = ref.watch(briefingPollProvider);
  for (var i = 0; i < poll.tries; i++) {
    final b = await backend.briefing(sessionId);
    if (b != null) {
      if (b.readByCaregiverAt == null) {
        unawaited(backend.markBriefingRead(sessionId).catchError((_) {}));
      }
      yield b;
      return;
    }
    yield null;
    await Future<void>.delayed(poll.every);
  }
  throw const BriefingTimeout();
});

class BriefingTimeout implements Exception {
  const BriefingTimeout();

  @override
  String toString() => '브리핑이 아직 없어요. 잠시 뒤에 다시 열어 주세요.';
}
