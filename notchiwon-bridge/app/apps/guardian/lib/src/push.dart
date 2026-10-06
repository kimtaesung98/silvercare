import 'dart:async';

import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

/// 서버가 보내는 알림 한 건. 위급 알림이면 [escalationId]가, 하루 소식이면
/// [elderId]와 [date]가 들어 있습니다.
class Alert {
  const Alert({
    required this.type,
    required this.title,
    required this.body,
    this.escalationId,
    this.elderId,
    this.date,
  });

  /// 서버 `Data["type"]`: `escalation` 또는 `digest`.
  final String type;
  final String title;
  final String body;
  final String? escalationId;
  final String? elderId;
  final String? date;

  bool get isEscalation => type == 'escalation' && escalationId != null;

  static Alert? of(RemoteMessage m) {
    final type = m.data['type'];
    if (type is! String) return null;
    return Alert(
      type: type,
      title: m.notification?.title ?? (type == 'digest' ? '하루 소식' : '위급 알림'),
      body: m.notification?.body ?? '',
      escalationId: m.data['escalationId'] as String?,
      elderId: m.data['elderId'] as String?,
      date: m.data['date'] as String?,
    );
  }
}

/// 위급 알림과 하루 소식 푸시. 알림을 누르면 [onOpen]에, 앱을 보고 있는 중에
/// 알림이 오면 [onAlert]에 옵니다.
abstract interface class Push {
  Future<void> start({
    required Future<void> Function(String fcmToken) register,
    required void Function(Alert alert) onOpen,
    required void Function(Alert alert) onAlert,
  });
}

/// Firebase 설정이 없을 때. 위급 알림은 홈 화면의 미확인 목록으로만 봅니다.
class NoPush implements Push {
  const NoPush();

  @override
  Future<void> start({
    required Future<void> Function(String fcmToken) register,
    required void Function(Alert alert) onOpen,
    required void Function(Alert alert) onAlert,
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
    required void Function(Alert alert) onOpen,
    required void Function(Alert alert) onAlert,
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
      final a = Alert.of(msg);
      if (a != null) onAlert(a);
    });
    FirebaseMessaging.onMessageOpenedApp.listen((msg) {
      final a = Alert.of(msg);
      if (a != null) onOpen(a);
    });
    final initial = await m.getInitialMessage();
    final a = initial == null ? null : Alert.of(initial);
    if (a != null) onOpen(a);
  }
}
