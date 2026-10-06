import 'dart:async';

import 'package:api_client/api_client.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';

import 'backend.dart';
import 'providers.dart';

/// 휴대폰이 잡은 위치 하나.
class Fix {
  const Fix({
    required this.latitude,
    required this.longitude,
    required this.at,
    this.accuracyM,
  });

  final double latitude;
  final double longitude;
  final double? accuracyM;
  final DateTime at;
}

/// 위치를 받을 수 있는지. 권한이 없으면 이동을 시작하지 않습니다.
enum LocationAccess { granted, denied, serviceOff }

abstract interface class PositionSource {
  Future<LocationAccess> requestAccess();

  /// 위치가 바뀔 때마다 하나씩. 듣는 동안 Android에서는 위치 알림이 떠 있습니다.
  Stream<Fix> watch();
}

/// `geolocator`로 위치를 받습니다. Android에서는 위치 유형 Foreground Service를
/// 띄워 화면이 꺼지거나 다른 앱(카카오맵 길안내)을 써도 계속 받습니다.
class GeolocatorSource implements PositionSource {
  const GeolocatorSource();

  @override
  Future<LocationAccess> requestAccess() async {
    if (!await Geolocator.isLocationServiceEnabled()) {
      return LocationAccess.serviceOff;
    }
    var p = await Geolocator.checkPermission();
    if (p == LocationPermission.denied) {
      p = await Geolocator.requestPermission();
    }
    return p == LocationPermission.always || p == LocationPermission.whileInUse
        ? LocationAccess.granted
        : LocationAccess.denied;
  }

  @override
  Stream<Fix> watch() =>
      Geolocator.getPositionStream(
        locationSettings: AndroidSettings(
          accuracy: LocationAccuracy.high,
          distanceFilter: 0,
          intervalDuration: const Duration(seconds: 5),
          foregroundNotificationConfig: const ForegroundNotificationConfig(
            notificationTitle: '어르신 댁으로 이동 중',
            notificationText: '도착 시간을 어르신 태블릿에 알리려고 위치를 보내고 있어요',
            notificationChannelName: '이동 중 위치',
            enableWakeLock: true,
            setOngoing: true,
          ),
        ),
      ).map(
        (p) => Fix(
          latitude: p.latitude,
          longitude: p.longitude,
          accuracyM: p.accuracy,
          at: p.timestamp,
        ),
      );
}

/// 지금 이동 중인 방문 하나의 상태.
class Trip {
  const Trip({
    required this.visitId,
    this.lastFix,
    this.etaMinutes,
    this.sessionStarted = false,
    this.lastSentAt,
    this.error,
  });

  final String visitId;
  final Fix? lastFix;

  /// 서버가 마지막으로 계산한 남은 분.
  final int? etaMinutes;

  /// 어르신 태블릿에서 픽업 대기 대화가 시작되었는지.
  final bool sessionStarted;
  final DateTime? lastSentAt;

  /// 마지막 전송이 실패한 이유. 다음 전송이 성공하면 지워집니다.
  final String? error;

  Trip copyWith({
    Fix? lastFix,
    int? etaMinutes,
    bool? sessionStarted,
    DateTime? lastSentAt,
    String? error,
    bool clearError = false,
  }) => Trip(
    visitId: visitId,
    lastFix: lastFix ?? this.lastFix,
    etaMinutes: etaMinutes ?? this.etaMinutes,
    sessionStarted: sessionStarted ?? this.sessionStarted,
    lastSentAt: lastSentAt ?? this.lastSentAt,
    error: clearError ? null : (error ?? this.error),
  );
}

/// 출발하면 [reportIntervalProvider] 간격(10초)으로 가장 최근 위치를 서버에 보냅니다.
/// 한 번에 한 방문만 이동합니다. 다른 방문으로 출발하면 앞의 이동은 멈춥니다.
class TripController extends Notifier<Trip?> {
  StreamSubscription<Fix>? _positions;
  Timer? _timer;
  Fix? _latest;
  bool _sending = false;

  @override
  Trip? build() {
    ref.onDispose(_cancel);
    return null;
  }

  /// 위치 권한을 받고 이동을 시작합니다. 권한이 없으면 그 이유를 돌려줍니다.
  Future<LocationAccess> start(String visitId) async {
    final source = ref.read(positionSourceProvider);
    final access = await source.requestAccess();
    if (access != LocationAccess.granted) return access;

    _cancel();
    state = Trip(visitId: visitId);
    _positions = source.watch().listen((fix) {
      final first = _latest == null;
      _latest = fix;
      state = state?.copyWith(lastFix: fix);
      if (first) unawaited(_send());
    }, onError: (Object e) => state = state?.copyWith(error: '위치를 받지 못했어요'));
    _timer = Timer.periodic(ref.read(reportIntervalProvider), (_) => _send());
    return access;
  }

  void stop() {
    _cancel();
    state = null;
  }

  void _cancel() {
    _positions?.cancel();
    _positions = null;
    _timer?.cancel();
    _timer = null;
    _latest = null;
  }

  Future<void> _send() async {
    final trip = state;
    final fix = _latest;
    if (trip == null || fix == null || _sending) return;
    _sending = true;
    try {
      final r = await ref
          .read(backendProvider)
          .sendLocation(
            trip.visitId,
            LocationUpdate(
              latitude: fix.latitude,
              longitude: fix.longitude,
              accuracyM: fix.accuracyM,
              recordedAt: fix.at.toUtc(),
            ),
          );
      if (state?.visitId != trip.visitId) return;
      state = state?.copyWith(
        etaMinutes: r.etaMinutes,
        sessionStarted: r.sessionStarted || trip.sessionStarted,
        lastSentAt: DateTime.now(),
        clearError: true,
      );
    } on VisitClosedException {
      if (state?.visitId == trip.visitId) stop();
    } on SignedOutException {
      stop();
    } on Object {
      if (state?.visitId == trip.visitId) {
        state = state?.copyWith(error: '위치를 보내지 못했어요. 다시 시도하는 중');
      }
    } finally {
      _sending = false;
    }
  }
}
