import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'backend.dart';
import 'providers.dart';
import 'screens/escalation_screen.dart';
import 'screens/home_screen.dart';
import 'screens/login_screen.dart';
import 'screens/visit_screen.dart';

/// 조무사 앱. 로그인하지 않았으면 로그인 화면으로 보냅니다.
class CaregiverApp extends ConsumerStatefulWidget {
  const CaregiverApp({super.key});

  @override
  ConsumerState<CaregiverApp> createState() => _CaregiverAppState();
}

class _CaregiverAppState extends ConsumerState<CaregiverApp> {
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
          path: '/visits/:id',
          builder: (_, s) => VisitScreen(visitId: s.pathParameters['id']!),
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
            onOpen: (id) => _router.push('/escalations/$id'),
            onAlert: _showAlert,
          )
          .catchError((Object e) {
            debugPrint('푸시를 시작하지 못했습니다: $e');
          }),
    );
  }

  /// 앱을 보고 있을 때 온 위급 알림. Android는 이때 시스템 알림을 띄우지 않으므로
  /// 화면 위에 빨간 띠로 보여주고 미확인 목록을 새로 읽습니다.
  void _showAlert(String escalationId, String title, String body) {
    ref.invalidate(openEscalationsProvider);
    final m = _messenger.currentState;
    if (m == null) return;
    m.clearMaterialBanners();
    m.showMaterialBanner(
      MaterialBanner(
        backgroundColor: Colors.red.shade700,
        leading: const Icon(Icons.warning_amber_rounded, color: Colors.white),
        content: Text(
          body.isEmpty ? title : '$title\n$body',
          style: const TextStyle(color: Colors.white, fontSize: 16),
        ),
        actions: [
          TextButton(
            onPressed: () {
              m.hideCurrentMaterialBanner();
              _router.push('/escalations/$escalationId');
            },
            child: const Text('보기', style: TextStyle(color: Colors.white)),
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
      title: '노치원 조무사',
      scaffoldMessengerKey: _messenger,
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF2E7D32),
        useMaterial3: true,
      ),
      routerConfig: _router,
    );
  }
}
