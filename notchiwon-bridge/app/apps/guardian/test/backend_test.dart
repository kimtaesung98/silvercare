import 'dart:convert';

import 'package:api_client/api_client.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:guardian/src/backend.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

void main() {
  late String? token;
  late int signedOut;
  late List<http.Request> requests;

  RestBackend backend(int status, Object body) => RestBackend(
    serverUrl: Uri.parse('http://server/'),
    token: () => token,
    onSignedOut: () => signedOut++,
    client: MockClient((r) async {
      requests.add(r);
      return http.Response(
        jsonEncode(body),
        status,
        headers: {'content-type': 'application/json; charset=utf-8'},
      );
    }),
  );

  setUp(() {
    token = 'tok-1';
    signedOut = 0;
    requests = [];
  });

  test('요청마다 지금 토큰을 붙인다', () async {
    final b = backend(200, {'escalations': <Object>[]});
    await b.openEscalations();
    token = 'tok-2';
    await b.openEscalations();
    expect(requests.map((r) => r.headers['Authorization']), [
      'Bearer tok-1',
      'Bearer tok-2',
    ]);
    expect(
      requests.first.url.toString(),
      'http://server/guardian/escalations/open',
    );
  });

  test('401이면 로그아웃시키고 SignedOutException', () async {
    final b = backend(401, {'code': 'UNAUTHORIZED', 'message': 'expired'});
    await expectLater(b.me(), throwsA(isA<SignedOutException>()));
    expect(signedOut, 1);
  });

  test('토큰이 없으면 서버에 묻지 않는다', () async {
    token = null;
    await expectLater(
      backend(200, {}).digests('e1'),
      throwsA(isA<SignedOutException>()),
    );
    expect(requests, isEmpty);
  });

  test('로그인 응답을 Credentials로', () async {
    token = null;
    final c = await backend(200, {
      'accessToken': 'new',
      'expiresAt': '2026-10-06T18:00:00Z',
      'guardian': {'id': '6b3f7a9e-1c2d-4e5f-8a9b-0c1d2e3f4a5b', 'name': '김보호'},
      'elders': [
        {'id': '0c1d2e3f-4a5b-4e5f-8a9b-6b3f7a9e1c2d', 'name': '김순자'},
      ],
    }).login('kim', 'pw');
    expect(c.accessToken, 'new');
    expect(c.guardianName, '김보호');
    expect(c.elders.single.name, '김순자');
    expect(c.expiresAt, DateTime.utc(2026, 10, 6, 18));
    expect(jsonDecode(requests.single.body), {
      'loginId': 'kim',
      'password': 'pw',
    });
    expect(requests.single.url.path, '/auth/guardian/login');
  });

  test('하루 소식과 설정을 읽고 쓴다', () async {
    final list = await backend(200, {
      'digests': [
        {
          'elderId': '0c1d2e3f-4a5b-4e5f-8a9b-6b3f7a9e1c2d',
          'date': '2026-10-06',
          'summaryText': '큰아들 이야기를 하셨어요.',
          'emotionFlag': null,
          'sessionCount': 2,
          'escalationCount': 0,
          'generatedAt': '2026-10-06T12:00:00Z',
          'sentAt': null,
        },
      ],
    }).digests('e1');
    expect(list.single.summaryText, '큰아들 이야기를 하셨어요.');
    // 서버는 날짜만 보내므로 기기 시간대의 그 날 자정으로 읽힙니다.
    expect(list.single.date, DateTime(2026, 10, 6));
    expect(requests.single.url.path, '/guardian/elders/e1/daily-digests');

    requests = [];
    final saved =
        await backend(200, {
          'enabled': true,
          'checkInTimes': ['10:00'],
          'bedtimeStart': '21:00',
          'bedtimeEnd': '07:00',
          'dailyTokenLimit': null,
          'timeZone': 'Asia/Seoul',
        }).saveSchedule(
          'e1',
          CompanionSchedule(
            enabled: true,
            checkInTimes: ['10:00'],
            bedtimeStart: '21:00',
            bedtimeEnd: '07:00',
            dailyTokenLimit: null,
            timeZone: 'Asia/Seoul',
          ),
        );
    expect(saved.checkInTimes, ['10:00']);
    expect(requests.single.method, 'PUT');
    expect(requests.single.url.path, '/guardian/elders/e1/companion-schedule');
  });

  test('푸시 토큰에 기기 이름을 붙여 보낸다', () async {
    await backend(204, {}).registerPushToken('fcm-1');
    expect(jsonDecode(requests.single.body), {
      'fcmToken': 'fcm-1',
      'label': '보호자 휴대폰',
    });
  });
}
