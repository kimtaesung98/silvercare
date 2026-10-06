import 'dart:convert';

import 'package:api_client/api_client.dart';
import 'package:http/http.dart' as http;

/// 로그인 결과. 토큰과 만료 시각, 보호자 이름, 돌보는 어르신 목록을 함께 보관합니다.
class Credentials {
  const Credentials({
    required this.accessToken,
    required this.expiresAt,
    required this.guardianName,
    required this.elders,
  });

  final String accessToken;
  final DateTime expiresAt;
  final String guardianName;

  /// 보호자가 돌보는 어르신. 보통 한 분입니다.
  final List<ElderSummary> elders;

  bool expiredAt(DateTime now) => !now.isBefore(expiresAt);
}

/// 토큰이 만료되었거나 거절되어 다시 로그인해야 합니다.
class SignedOutException implements Exception {
  const SignedOutException();

  @override
  String toString() => '다시 로그인해 주세요';
}

/// 보호자 앱이 쓰는 서버 기능. 화면과 테스트는 생성된 클라이언트 대신 이것만 봅니다.
abstract interface class Backend {
  Future<Credentials> login(String loginId, String password);
  Future<GuardianContext> me();
  Future<List<Escalation>> openEscalations();
  Future<Escalation> escalation(String escalationId);
  Future<Escalation> ackEscalation(String escalationId);
  Future<List<DailyDigest>> digests(String elderId);
  Future<CompanionSchedule> schedule(String elderId);
  Future<CompanionSchedule> saveSchedule(String elderId, CompanionSchedule s);
  Future<void> registerPushToken(String fcmToken);
}

/// 생성된 REST 클라이언트로 서버를 부릅니다.
class RestBackend implements Backend {
  RestBackend({
    required Uri serverUrl,
    required this.token,
    required this.onSignedOut,
    http.Client? client,
  }) {
    _client = ApiClient(
      basePath: serverUrl.toString().replaceAll(RegExp(r'/$'), ''),
      authentication: _LiveBearer(token),
    );
    if (client != null) _client.client = client;
  }

  late final ApiClient _client;

  /// 지금 로그인한 토큰. 로그인하지 않았으면 null입니다.
  final String? Function() token;

  /// 서버가 토큰을 거절했을 때 부릅니다.
  final void Function() onSignedOut;

  late final _auth = AuthApi(_client);
  late final _guardian = GuardianApi(_client);

  /// 401은 로그아웃으로 바꿉니다.
  Future<T> _call<T>(Future<T?> Function() f) async {
    if (token() == null) throw const SignedOutException();
    try {
      final v = await f();
      if (v == null) throw StateError('서버가 빈 응답을 보냈습니다');
      return v;
    } on ApiException catch (e) {
      if (e.code == 401) {
        onSignedOut();
        throw const SignedOutException();
      }
      rethrow;
    }
  }

  @override
  Future<Credentials> login(String loginId, String password) async {
    final r = await _auth.loginGuardian(
      CaregiverLoginRequest(loginId: loginId, password: password),
    );
    if (r == null) throw StateError('서버가 빈 응답을 보냈습니다');
    return Credentials(
      accessToken: r.accessToken,
      expiresAt: r.expiresAt,
      guardianName: r.guardian.name,
      elders: r.elders,
    );
  }

  @override
  Future<GuardianContext> me() => _call(_guardian.getGuardianContext);

  @override
  Future<List<Escalation>> openEscalations() async =>
      (await _call(_guardian.listGuardianOpenEscalations)).escalations;

  @override
  Future<Escalation> escalation(String escalationId) =>
      _call(() => _guardian.getGuardianEscalation(escalationId));

  @override
  Future<Escalation> ackEscalation(String escalationId) =>
      _call(() => _guardian.ackGuardianEscalation(escalationId));

  @override
  Future<List<DailyDigest>> digests(String elderId) async =>
      (await _call(() => _guardian.listDailyDigests(elderId))).digests;

  @override
  Future<CompanionSchedule> schedule(String elderId) =>
      _call(() => _guardian.getGuardianCompanionSchedule(elderId));

  @override
  Future<CompanionSchedule> saveSchedule(String elderId, CompanionSchedule s) =>
      _call(() => _guardian.putGuardianCompanionSchedule(elderId, s));

  @override
  Future<void> registerPushToken(String fcmToken) => _call<bool>(() async {
    await _guardian.putGuardianFcmToken(
      FcmTokenRegistration(fcmToken: fcmToken, label: '보호자 휴대폰'),
    );
    return true;
  });
}

/// 요청할 때마다 현재 토큰을 읽는 Bearer 인증. 다시 로그인해도 클라이언트를 새로 만들 필요가 없습니다.
class _LiveBearer implements Authentication {
  _LiveBearer(this._token);

  final String? Function() _token;

  @override
  Future<void> applyToParams(
    List<QueryParam> queryParams,
    Map<String, String> headerParams,
  ) async {
    final t = _token();
    if (t != null) headerParams['Authorization'] = 'Bearer $t';
  }
}

/// 오류 응답 본문(`ApiError`)의 `code`. JSON이 아니면 null입니다.
String? errorCode(ApiException e) {
  try {
    final body = jsonDecode(e.message ?? '');
    return body is Map<String, dynamic> ? body['code'] as String? : null;
  } on FormatException {
    return null;
  }
}
