import 'dart:convert';

import 'package:api_client/api_client.dart';
import 'package:caregiver/src/backend.dart';
import 'package:flutter_test/flutter_test.dart';
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
    final b = backend(200, {'visits': <Object>[]});
    await b.todayVisits();
    token = 'tok-2';
    await b.todayVisits();
    expect(requests.map((r) => r.headers['Authorization']), [
      'Bearer tok-1',
      'Bearer tok-2',
    ]);
    expect(requests.first.url.toString(), 'http://server/visits/today');
  });

  test('401이면 로그아웃시키고 SignedOutException', () async {
    final b = backend(401, {'code': 'UNAUTHORIZED', 'message': 'expired'});
    await expectLater(b.todayVisits(), throwsA(isA<SignedOutException>()));
    expect(signedOut, 1);
  });

  test('토큰이 없으면 서버에 묻지 않는다', () async {
    token = null;
    await expectLater(
      backend(200, {}).openEscalations(),
      throwsA(isA<SignedOutException>()),
    );
    expect(requests, isEmpty);
  });

  test('BRIEFING_NOT_READY는 null, 다른 404는 오류', () async {
    expect(
      await backend(404, {
        'code': 'BRIEFING_NOT_READY',
        'message': '',
      }).briefing('s1'),
      isNull,
    );
    await expectLater(
      backend(404, {'code': 'NOT_FOUND', 'message': ''}).briefing('s1'),
      throwsA(isA<ApiException>().having((e) => e.code, 'code', 404)),
    );
  });

  test('VISIT_CLOSED는 VisitClosedException', () async {
    await expectLater(
      backend(409, {'code': 'VISIT_CLOSED', 'message': ''}).sendLocation(
        'v1',
        LocationUpdate(
          latitude: 37,
          longitude: 127,
          recordedAt: DateTime.now(),
        ),
      ),
      throwsA(isA<VisitClosedException>()),
    );
  });

  test('로그인 응답을 Credentials로', () async {
    token = null;
    final c = await backend(200, {
      'accessToken': 'new',
      'expiresAt': '2026-10-06T18:00:00Z',
      'caregiver': {
        'id': '6b3f7a9e-1c2d-4e5f-8a9b-0c1d2e3f4a5b',
        'name': '이조무',
        'centerId': '0c1d2e3f-4a5b-4e5f-8a9b-6b3f7a9e1c2d',
      },
    }).login('lee', 'pw');
    expect(c.accessToken, 'new');
    expect(c.caregiverName, '이조무');
    expect(c.expiresAt, DateTime.utc(2026, 10, 6, 18));
    expect(jsonDecode(requests.single.body), {
      'loginId': 'lee',
      'password': 'pw',
    });
  });
}
