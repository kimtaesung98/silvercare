import 'package:api_client/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'backend.dart';
import 'credential_store.dart';
import 'push.dart';

/// `main`에서 실제 구현으로, 테스트에서 가짜로 바꿔 끼웁니다.
final backendProvider = Provider<Backend>(
  (ref) => throw UnimplementedError('backendProvider'),
);
final credentialStoreProvider = Provider<CredentialStore>(
  (ref) => MemoryCredentialStore(),
);

/// 앱을 켤 때 보관소에서 읽은 로그인 정보.
final initialCredentialsProvider = Provider<Credentials?>((ref) => null);
final pushProvider = Provider<Push>((ref) => const NoPush());

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
    await ref.read(credentialStoreProvider).clear();
    state = null;
  }
}

final authProvider = NotifierProvider<Auth, Credentials?>(Auth.new);

/// 돌보는 어르신. 로그인할 때 받은 목록을 서버에서 다시 확인합니다.
final eldersProvider = FutureProvider.autoDispose<List<ElderSummary>>((
  ref,
) async {
  final me = await ref.watch(backendProvider).me();
  return me.elders;
});

final openEscalationsProvider = FutureProvider.autoDispose<List<Escalation>>(
  (ref) => ref.watch(backendProvider).openEscalations(),
);

final escalationProvider = FutureProvider.autoDispose
    .family<Escalation, String>(
      (ref, id) => ref.watch(backendProvider).escalation(id),
    );

final digestsProvider = FutureProvider.autoDispose
    .family<List<DailyDigest>, String>(
      (ref, elderId) => ref.watch(backendProvider).digests(elderId),
    );

final scheduleProvider = FutureProvider.autoDispose
    .family<CompanionSchedule, String>(
      (ref, elderId) => ref.watch(backendProvider).schedule(elderId),
    );
