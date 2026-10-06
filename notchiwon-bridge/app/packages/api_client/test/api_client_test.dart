import 'package:flutter_test/flutter_test.dart';

import 'package:api_client/api_client.dart';

void main() {
  test('decodes a visit from the server JSON shape', () {
    final visit = Visit.fromJson({
      'id': '0b7c2a4e-3f1d-4a8e-9a51-6f4f3c2d1e00',
      'elder': {'id': '7d1e2f3a-4b5c-4d6e-8f90-a1b2c3d4e5f6', 'name': '김순자'},
      'scheduledTime': '2026-10-06T07:30:00Z',
      'etaCurrent': null,
      'status': 'EN_ROUTE',
      'sessionId': null,
    });

    expect(visit, isNotNull);
    expect(visit!.elder.name, '김순자');
    expect(visit.status, VisitStatus.EN_ROUTE);
    expect(visit.scheduledTime.toUtc(), DateTime.utc(2026, 10, 6, 7, 30));
    expect(visit.etaCurrent, isNull);
  });

  test('encodes a location update with the field names the server expects', () {
    final update = LocationUpdate(
      latitude: 37.5665,
      longitude: 126.978,
      recordedAt: DateTime.utc(2026, 10, 6, 7, 20),
    );

    final json = update.toJson();
    expect(json['latitude'], 37.5665);
    expect(json['longitude'], 126.978);
    expect(json['recordedAt'], '2026-10-06T07:20:00.000Z');
  });

  test('round-trips a companion schedule with nullable fields', () {
    final schedule = CompanionSchedule.fromJson({
      'enabled': true,
      'checkInTimes': ['15:30', '19:00'],
      'bedtimeStart': '21:00',
      'bedtimeEnd': '07:00',
      'timeZone': 'Asia/Seoul',
      'dailyTokenLimit': null,
    });

    expect(schedule!.checkInTimes, ['15:30', '19:00']);
    expect(schedule.dailyTokenLimit, isNull);
    expect(schedule.toJson()['bedtimeEnd'], '07:00');
  });
}
