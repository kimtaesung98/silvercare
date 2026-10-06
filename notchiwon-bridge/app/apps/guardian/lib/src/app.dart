import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'backend.dart';
import 'providers.dart';
import 'push.dart';
import 'screens/elder_screen.dart';
import 'screens/escalation_screen.dart';
import 'screens/home_screen.dart';
import 'screens/login_screen.dart';
import 'screens/settings_screen.dart';

/// 보호자 앱. 로그인하지 않았으면 로그인 화면으로 보냅니다.
class GuardianApp extends ConsumerStatefulWidget {
  const GuardianApp({super.key});

  @override
  ConsumerState<GuardianApp> createState() => _GuardianAppState();
}

class _GuardianAppState extends ConsumerState<GuardianApp> {
  final _messenger = GlobalKey<ScaffoldMessengerState>();
  final _signedIn = ValueNotifier(false);
  late final GoRouter _router;

  @override
  void initState() {
    super.initState();
    _signedIn.value = ref.read(authProvider) != null;
    _router = GoRouter(
      refreshListenable: _signedIn,
      redirect: (context, state) {
        final atLogin = state.matchedLocation == '/login';
        if (!_signedIn.value) return atLogin ? null : '/login';
        return atLogin ? '/' : null;
      },
      routes: [
        GoRoute(path: '/', builder: (_, _) => const HomeScreen()),
        GoRoute(path: '/login', builder: (_, _) => const LoginScreen()),
        GoRoute(
          path: '/elders/:id',
          builder: (_, s) => ElderScreen(elderId: s.pathParameters['id']!),
        ),
        GoRoute(
          path: '/elders/:id/settings',
          builder: (_, s) => SettingsScreen(elderId: s.pathParameters['id']!),
        ),
        GoRoute(
          path: '/escalations/:id',
          builder: (_, s) =>
              EscalationScreen(escalationId: s.pathParameters['id']!),
        ),
      ],
    );
    if (_signedIn.value) _startPush();
  }

  @override
  void dispose() {
    _router.dispose();
    _signedIn.dispose();
    super.dispose();
  }

  void _startPush() {
    final backend = ref.read(backendProvider);
    unawaited(
      ref
          .read(pushProvider)
          .start(
            register: (token) async {
              try {
                await backend.registerPushToken(token);
              } on SignedOutException {
                // 로그인 화면으로 이미 넘어갑니다.
              }
            },
            onOpen: _open,
            onAlert: _showAlert,
          )
          .catchError((Object e) {
            debugPrint('푸시를 시작하지 못했습니다: $e');
          }),
    );
  }

  /// 알림을 눌러 앱이 열렸을 때 갈 화면.
  void _open(Alert a) {
    if (a.isEscalation) {
      _router.push('/escalations/${a.escalationId}');
      return;
    }
    final elderId = a.elderId;
    if (elderId != null) {
      ref.invalidate(digestsProvider(elderId));
      _router.push('/elders/$elderId');
    }
  }

  /// 앱을 보고 있을 때 온 알림. Android는 이때 시스템 알림을 띄우지 않으므로
  /// 화면 위에 띠로 보여줍니다. 위급 알림은 빨간 띠로, 하루 소식은 보통 띠로요.
  void _showAlert(Alert a) {
    if (a.isEscalation) {
      ref.invalidate(openEscalationsProvider);
    } else if (a.elderId != null) {
      ref.invalidate(digestsProvider(a.elderId!));
    }
    final m = _messenger.currentState;
    if (m == null) return;
    final urgent = a.isEscalation;
    final scheme = Theme.of(context).colorScheme;
    m.clearMaterialBanners();
    m.showMaterialBanner(
      MaterialBanner(
        backgroundColor: urgent
            ? Colors.red.shade700
            : scheme.secondaryContainer,
        leading: Icon(
          urgent ? Icons.warning_amber_rounded : Icons.mail_outline,
          color: urgent ? Colors.white : scheme.onSecondaryContainer,
        ),
        content: Text(
          a.body.isEmpty ? a.title : '${a.title}\n${a.body}',
          style: TextStyle(
            color: urgent ? Colors.white : scheme.onSecondaryContainer,
            fontSize: 16,
          ),
        ),
        actions: [
          TextButton(
            onPressed: () {
              m.hideCurrentMaterialBanner();
              _open(a);
            },
            child: Text(
              '보기',
              style: TextStyle(
                color: urgent ? Colors.white : scheme.onSecondaryContainer,
              ),
            ),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(authProvider, (prev, next) {
      _signedIn.value = next != null;
      if (prev == null && next != null) _startPush();
      if (next == null) _messenger.currentState?.clearMaterialBanners();
    });
    return MaterialApp.router(
      title: '노치원 보호자',
      scaffoldMessengerKey: _messenger,
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF00695C),
        useMaterial3: true,
      ),
      routerConfig: _router,
    );
  }
}
