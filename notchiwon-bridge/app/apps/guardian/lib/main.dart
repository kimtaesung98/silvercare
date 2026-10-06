import 'package:firebase_core/firebase_core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'src/app.dart';
import 'src/backend.dart';
import 'src/credential_store.dart';
import 'src/providers.dart';
import 'src/push.dart';

/// 서버 주소. 빌드할 때 넣습니다: `--dart-define=SERVER_URL=https://...`.
/// 기본값은 안드로이드 에뮬레이터에서 보이는 개발 서버입니다.
const serverUrl = String.fromEnvironment(
  'SERVER_URL',
  defaultValue: 'http://10.0.2.2:8000',
);

/// Firebase 앱 설정(Firebase 콘솔 > 프로젝트 설정 > Android 앱). 비워 두면
/// 푸시 없이 동작하고, 위급 알림은 홈 화면의 미확인 목록으로만 봅니다.
const _firebaseApiKey = String.fromEnvironment('FIREBASE_API_KEY');
const _firebaseAppId = String.fromEnvironment('FIREBASE_APP_ID');
const _firebaseSenderId = String.fromEnvironment('FIREBASE_SENDER_ID');
const _firebaseProjectId = String.fromEnvironment('FIREBASE_PROJECT_ID');

Push _push() {
  if (_firebaseApiKey.isEmpty ||
      _firebaseAppId.isEmpty ||
      _firebaseSenderId.isEmpty ||
      _firebaseProjectId.isEmpty) {
    return const NoPush();
  }
  return FirebasePush(
    const FirebaseOptions(
      apiKey: _firebaseApiKey,
      appId: _firebaseAppId,
      messagingSenderId: _firebaseSenderId,
      projectId: _firebaseProjectId,
    ),
  );
}

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final store = SecureCredentialStore();
  final saved = await store.read();

  late final ProviderContainer container;
  container = ProviderContainer(
    retry: (_, _) => null,
    overrides: [
      credentialStoreProvider.overrideWithValue(store),
      initialCredentialsProvider.overrideWithValue(saved),
      pushProvider.overrideWithValue(_push()),
      backendProvider.overrideWithValue(
        RestBackend(
          serverUrl: Uri.parse(serverUrl),
          token: () => container.read(authProvider)?.accessToken,
          onSignedOut: () => container.read(authProvider.notifier).signOut(),
        ),
      ),
    ],
  );
  runApp(
    UncontrolledProviderScope(container: container, child: const GuardianApp()),
  );
}
