import 'dart:async';

import 'package:api_client/api_client.dart';
import 'package:caregiver/src/backend.dart';
import 'package:caregiver/src/credential_store.dart';
import 'package:caregiver/src/map_tiles.dart';
import 'package:caregiver/src/providers.dart';
import 'package:caregiver/src/push.dart';
import 'package:caregiver/src/trip.dart';
import 'package:flutter_riverpod/misc.dart';

final now = DateTime.utc(2026, 10, 6, 6);

Visit visit({
  String id = 'v1',
  String elder = '김순자',
  VisitStatus status = VisitStatus.SCHEDULED,
  String? sessionId,
  Place? destination,
}) => Visit(
  id: id,
  elder: ElderSummary(id: 'e-$id', name: elder),
  scheduledTime: now.add(const Duration(minutes: 30)),
  status: status,
  sessionId: sessionId,
  destination:
      destination ??
      Place(latitude: 37.5725, longitude: 126.9769, address: '서울 종로구 세종대로 175'),
);

Escalation escalation({
  String id = 'x1',
  String? visitId = 'v1',
  DateTime? acknowledgedAt,
}) => Escalation(
  id: id,
  sessionId: 's1',
  elder: ElderSummary(id: 'e-v1', name: '김순자'),
  triggerType: EscalationTriggerType.PAIN_COMPLAINT,
  source_: EscalationSource_Enum.RULE,
  utteranceText: '다리가 아파',
  createdAt: now,
  acknowledgedAt: acknowledgedAt,
  visitId: visitId,
);

final credentials = Credentials(
  accessToken: 'tok',
  expiresAt: DateTime.utc(2100),
  caregiverName: '이조무',
);

/// 메모리 위의 서버.
class FakeBackend implements Backend {
  final visits = <String, Visit>{};
  final escalations = <String, Escalation>{};
  final sent = <(String, LocationUpdate)>[];
  final arrived = <String>[];
  final pushTokens = <String>[];
  final read = <String>[];

  /// 브리핑을 몇 번 물어본 뒤에 내줄지.
  int briefingAfter = 0;
  int briefingCalls = 0;
  Briefing? briefingValue;
  LocationUpdateResult Function(String visitId)? onLocation;
  Object? locationError;

  @override
  Future<Credentials> login(String loginId, String password) async {
    if (password != 'pw') throw ApiException(401, '{"code":"UNAUTHORIZED"}');
    return Credentials(
      accessToken: 'tok',
      expiresAt: DateTime.now().add(const Duration(hours: 12)),
      caregiverName: '이조무',
    );
  }

  @override
  Future<List<Visit>> todayVisits() async => visits.values.toList();

  @override
  Future<Visit> visit(String visitId) async => visits[visitId]!;

  @override
  Future<LocationUpdateResult> sendLocation(
    String visitId,
    LocationUpdate fix,
  ) async {
    sent.add((visitId, fix));
    if (locationError case final e?) throw e;
    return onLocation?.call(visitId) ??
        LocationUpdateResult(
          visitId: visitId,
          status: VisitStatus.EN_ROUTE,
          etaMinutes: 12,
          sessionStarted: false,
        );
  }

  @override
  Future<Visit> arrive(String visitId) async {
    arrived.add(visitId);
    final v = visits[visitId]!;
    v.status = VisitStatus.COMPLETED;
    v.sessionId ??= 's1';
    return v;
  }

  @override
  Future<Briefing?> briefing(String sessionId) async {
    briefingCalls++;
    return briefingCalls > briefingAfter ? briefingValue : null;
  }

  @override
  Future<void> markBriefingRead(String sessionId) async => read.add(sessionId);

  @override
  Future<List<Escalation>> openEscalations() async =>
      escalations.values.where((e) => e.acknowledgedAt == null).toList();

  @override
  Future<Escalation> escalation(String escalationId) async =>
      escalations[escalationId]!;

  @override
  Future<Escalation> ackEscalation(String escalationId) async {
    final e = escalations[escalationId]!;
    e.acknowledgedAt ??= now.add(const Duration(minutes: 1));
    return e;
  }

  @override
  Future<void> registerPushToken(String fcmToken) async =>
      pushTokens.add(fcmToken);
}

/// 손으로 위치를 넣는 위치 소스.
class FakePositions implements PositionSource {
  FakePositions([this.access = LocationAccess.granted]);

  LocationAccess access;
  final controller = StreamController<Fix>.broadcast();

  @override
  Future<LocationAccess> requestAccess() async => access;

  @override
  Stream<Fix> watch() => controller.stream;

  void move(double lat, double lng, DateTime at) =>
      controller.add(Fix(latitude: lat, longitude: lng, at: at));
}

/// 시작하면 토큰을 등록하고, 테스트가 알림을 흉내 낼 수 있게 콜백을 잡아 둡니다.
class FakePush implements Push {
  void Function(String)? open;
  void Function(String, String, String)? alert;

  @override
  Future<void> start({
    required Future<void> Function(String fcmToken) register,
    required void Function(String escalationId) onOpen,
    required void Function(String escalationId, String title, String body)
    onAlert,
  }) async {
    open = onOpen;
    alert = onAlert;
    await register('fcm-token');
  }
}

List<Override> overrides({
  required FakeBackend backend,
  FakePositions? positions,
  FakePush? push,
  bool signedIn = true,
}) => [
  backendProvider.overrideWithValue(backend),
  credentialStoreProvider.overrideWithValue(
    MemoryCredentialStore(signedIn ? credentials : null),
  ),
  initialCredentialsProvider.overrideWithValue(signedIn ? credentials : null),
  positionSourceProvider.overrideWithValue(positions ?? FakePositions()),
  pushProvider.overrideWithValue(push ?? FakePush()),
  mapTilesProvider.overrideWithValue(null),
  briefingPollProvider.overrideWithValue((
    every: const Duration(seconds: 3),
    tries: 5,
  )),
];
