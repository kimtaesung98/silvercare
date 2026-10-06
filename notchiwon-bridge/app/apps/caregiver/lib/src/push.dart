import 'dart:async';

import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

/// 위급 알림 푸시. 알림을 누르면 [onOpen]에 위급 이벤트 ID가, 앱을 보고
/// 있는 중에 알림이 오면 [onAlert]에 제목과 본문이 옵니다.
abstract interface class Push {
  Future<void> start({
    required Future<void> Function(String fcmToken) register,
    required void Function(String escalationId) onOpen,
    required void Function(String escalationId, String title, String body)
    onAlert,
  });
}

/// Firebase 설정이 없을 때. 위급 알림은 홈 화면의 미확인 목록으로만 봅니다.
class NoPush implements Push {
  const NoPush();

  @override
  Future<void> start({
    required Future<void> Function(String fcmToken) register,
    required void Function(String escalationId) onOpen,
    required void Function(String escalationId, String title, String body)
    onAlert,
  }) async {}
}

/// Firebase Cloud Messaging. 설정은 `google-services.json` 대신
/// `--dart-define`으로 받은 [FirebaseOptions]를 씁니다.
class FirebasePush implements Push {
  FirebasePush(this.options);

  final FirebaseOptions options;
  bool _started = false;

  @override
  Future<void> start({
    required Future<void> Function(String fcmToken) register,
    required void Function(String escalationId) onOpen,
    required void Function(String escalationId, String title, String body)
    onAlert,
  }) async {
    if (Firebase.apps.isEmpty) {
      await Firebase.initializeApp(options: options);
    }
    final m = FirebaseMessaging.instance;
    await m.requestPermission();
    final token = await m.getToken();
    if (token != null) await register(token);
    if (_started) return;
    _started = true;

    m.onTokenRefresh.listen((t) => unawaited(register(t)));
    FirebaseMessaging.onMessage.listen((msg) {
      final id = msg.data['escalationId'];
      if (id is! String) return;
      onAlert(
        id,
        msg.notification?.title ?? '위급 알림',
        msg.notification?.body ?? '',
      );
    });
    FirebaseMessaging.onMessageOpenedApp.listen((msg) {
      final id = msg.data['escalationId'];
      if (id is String) onOpen(id);
    });
    final initial = await m.getInitialMessage();
    final id = initial?.data['escalationId'];
    if (id is String) onOpen(id);
  }
}
