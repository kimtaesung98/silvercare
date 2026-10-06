import 'dart:convert';

import 'package:api_client/api_client.dart';
import 'package:http/http.dart' as http;

/// 로그인 결과. 토큰과 만료 시각, 조무사 이름을 함께 보관합니다.
class Credentials {
  const Credentials({
    required this.accessToken,
    required this.expiresAt,
    required this.caregiverName,
  });

  final String accessToken;
  final DateTime expiresAt;
  final String caregiverName;

  bool expiredAt(DateTime now) => !now.isBefore(expiresAt);
}

/// 토큰이 만료되었거나 거절되어 다시 로그인해야 합니다.
class SignedOutException implements Exception {
  const SignedOutException();

  @override
  String toString() => '다시 로그인해 주세요';
}

/// 방문이 이미 끝나 위치나 도착을 받을 수 없습니다 (`VISIT_CLOSED`).
class VisitClosedException implements Exception {
  const VisitClosedException();

  @override
  String toString() => '이미 끝난 방문입니다';
}

/// 조무사 앱이 쓰는 서버 기능. 화면과 테스트는 생성된 클라이언트 대신 이것만 봅니다.
abstract interface class Backend {
  Future<Credentials> login(String loginId, String password);
  Future<List<Visit>> todayVisits();
  Future<Visit> visit(String visitId);
  Future<LocationUpdateResult> sendLocation(String visitId, LocationUpdate fix);
  Future<Visit> arrive(String visitId);

  /// 브리핑이 아직 만들어지지 않았으면 null입니다.
  Future<Briefing?> briefing(String sessionId);
  Future<void> markBriefingRead(String sessionId);
  Future<List<Escalation>> openEscalations();
  Future<Escalation> escalation(String escalationId);
  Future<Escalation> ackEscalation(String escalationId);
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
    final auth = _LiveBearer(token);
    _client = ApiClient(
      basePath: serverUrl.toString().replaceAll(RegExp(r'/$'), ''),
      authentication: auth,
    );
    if (client != null) _client.client = client;
  }

  late final ApiClient _client;

  /// 지금 로그인한 토큰. 로그인하지 않았으면 null입니다.
  final String? Function() token;

  /// 서버가 토큰을 거절했을 때 부릅니다.
  final void Function() onSignedOut;

  late final _auth = AuthApi(_client);
  late final _visits = VisitsApi(_client);
  late final _briefings = BriefingsApi(_client);
  late final _escalations = EscalationsApi(_client);
  late final _devices = DevicesApi(_client);

  /// 401은 로그아웃으로, 409 `VISIT_CLOSED`는 [VisitClosedException]으로 바꿉니다.
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
      if (e.code == 409 && _errorCode(e) == 'VISIT_CLOSED') {
        throw const VisitClosedException();
      }
      rethrow;
    }
  }

  @override
  Future<Credentials> login(String loginId, String password) async {
    final r = await _auth.loginCaregiver(
      CaregiverLoginRequest(loginId: loginId, password: password),
    );
    if (r == null) throw StateError('서버가 빈 응답을 보냈습니다');
    return Credentials(
      accessToken: r.accessToken,
      expiresAt: r.expiresAt,
      caregiverName: r.caregiver.name,
    );
  }

  @override
  Future<List<Visit>> todayVisits() async =>
      (await _call(_visits.listTodayVisits)).visits;

  @override
  Future<Visit> visit(String visitId) => _call(() => _visits.getVisit(visitId));

  @override
  Future<LocationUpdateResult> sendLocation(
    String visitId,
    LocationUpdate fix,
  ) => _call(() => _visits.postVisitLocation(visitId, fix));

  @override
  Future<Visit> arrive(String visitId) =>
      _call(() => _visits.arriveVisit(visitId));

  @override
  Future<Briefing?> briefing(String sessionId) async {
    try {
      return await _call(() => _briefings.getSessionBriefing(sessionId));
    } on ApiException catch (e) {
      if (e.code == 404 && _errorCode(e) == 'BRIEFING_NOT_READY') return null;
      rethrow;
    }
  }

  @override
  Future<void> markBriefingRead(String sessionId) =>
      _call(() => _briefings.markBriefingRead(sessionId));

  @override
  Future<List<Escalation>> openEscalations() async =>
      (await _call(_escalations.listOpenEscalations)).escalations;

  @override
  Future<Escalation> escalation(String escalationId) =>
      _call(() => _escalations.getEscalation(escalationId));

  @override
  Future<Escalation> ackEscalation(String escalationId) =>
      _call(() => _escalations.ackEscalation(escalationId));

  @override
  Future<void> registerPushToken(String fcmToken) => _call<bool>(() async {
    await _devices.putCaregiverFcmToken(
      FcmTokenRegistration(fcmToken: fcmToken, label: '조무사 휴대폰'),
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
String? _errorCode(ApiException e) {
  try {
    final body = jsonDecode(e.message ?? '');
    return body is Map<String, dynamic> ? body['code'] as String? : null;
  } on FormatException {
    return null;
  }
}
