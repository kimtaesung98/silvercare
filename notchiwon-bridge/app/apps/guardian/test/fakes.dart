import 'package:api_client/api_client.dart';
import 'package:flutter_riverpod/misc.dart';
import 'package:guardian/src/backend.dart';
import 'package:guardian/src/credential_store.dart';
import 'package:guardian/src/providers.dart';
import 'package:guardian/src/push.dart';

final now = DateTime.utc(2026, 10, 6, 12);
final elder = ElderSummary(id: 'e1', name: '김순자');

Escalation escalation({String id = 'x1', DateTime? acknowledgedAt}) =>
    Escalation(
      id: id,
      sessionId: 's1',
      elder: elder,
      triggerType: EscalationTriggerType.PAIN_COMPLAINT,
      source_: EscalationSource_Enum.RULE,
      utteranceText: '다리가 아파',
      createdAt: now,
      acknowledgedAt: acknowledgedAt,
      visitId: null,
    );

DailyDigest digest({
  DateTime? date,
  String summary = '큰아들 이야기를 즐겁게 하셨어요.',
  String? flag,
  int sessions = 1,
  int escalations = 0,
}) => DailyDigest(
  elderId: elder.id,
  date: date ?? DateTime.utc(2026, 10, 6),
  summaryText: summary,
  emotionFlag: flag,
  sessionCount: sessions,
  escalationCount: escalations,
  generatedAt: now,
  sentAt: now,
);

CompanionSchedule companionSchedule({
  bool enabled = true,
  List<String> checkIns = const ['10:00', '15:30'],
  String? bedStart = '21:00',
  String? bedEnd = '07:00',
  int? limit,
}) => CompanionSchedule(
  enabled: enabled,
  checkInTimes: checkIns,
  bedtimeStart: bedStart,
  bedtimeEnd: bedEnd,
  dailyTokenLimit: limit,
  timeZone: 'Asia/Seoul',
);

final credentials = Credentials(
  accessToken: 'tok',
  expiresAt: DateTime.utc(2100),
  guardianName: '김보호',
  elders: [elder],
);

/// 메모리 위의 서버.
class FakeBackend implements Backend {
  final escalations = <String, Escalation>{};
  final digestList = <DailyDigest>[];
  final pushTokens = <String>[];
  final saved = <CompanionSchedule>[];

  List<ElderSummary> elders = [elder];
  CompanionSchedule scheduleValue = companionSchedule();
  Object? digestError;

  @override
  Future<Credentials> login(String loginId, String password) async {
    if (password != 'pw') throw ApiException(401, '{"code":"UNAUTHORIZED"}');
    return Credentials(
      accessToken: 'tok',
      expiresAt: DateTime.now().add(const Duration(hours: 12)),
      guardianName: '김보호',
      elders: elders,
    );
  }

  @override
  Future<GuardianContext> me() async => GuardianContext(
    guardian: Guardian(id: 'g1', name: '김보호'),
    elders: elders,
  );

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
  Future<List<DailyDigest>> digests(String elderId) async {
    if (digestError != null) throw digestError!;
    return digestList;
  }

  @override
  Future<CompanionSchedule> schedule(String elderId) async => scheduleValue;

  @override
  Future<CompanionSchedule> saveSchedule(
    String elderId,
    CompanionSchedule s,
  ) async {
    saved.add(s);
    scheduleValue = s;
    return s;
  }

  @override
  Future<void> registerPushToken(String fcmToken) async =>
      pushTokens.add(fcmToken);
}

/// 시작하면 토큰을 등록하고, 테스트가 알림을 흉내 낼 수 있게 콜백을 잡아 둡니다.
class FakePush implements Push {
  void Function(Alert)? open;
  void Function(Alert)? alert;

  @override
  Future<void> start({
    required Future<void> Function(String fcmToken) register,
    required void Function(Alert alert) onOpen,
    required void Function(Alert alert) onAlert,
  }) async {
    open = onOpen;
    alert = onAlert;
    await register('fcm-token');
  }
}

List<Override> overrides({
  required FakeBackend backend,
  FakePush? push,
  bool signedIn = true,
}) => [
  backendProvider.overrideWithValue(backend),
  credentialStoreProvider.overrideWithValue(
    MemoryCredentialStore(signedIn ? credentials : null),
  ),
  initialCredentialsProvider.overrideWithValue(signedIn ? credentials : null),
  pushProvider.overrideWithValue(push ?? FakePush()),
];
