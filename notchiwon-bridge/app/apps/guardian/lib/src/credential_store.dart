import 'dart:convert';

import 'package:api_client/api_client.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'backend.dart';

/// 로그인 토큰을 앱을 다시 켜도 쓰도록 보관합니다.
abstract interface class CredentialStore {
  Future<Credentials?> read();
  Future<void> write(Credentials c);
  Future<void> clear();
}

/// Android Keystore로 암호화해 보관합니다.
class SecureCredentialStore implements CredentialStore {
  SecureCredentialStore([FlutterSecureStorage? storage])
    : _storage = storage ?? const FlutterSecureStorage();

  static const _key = 'guardian_credentials';
  final FlutterSecureStorage _storage;

  @override
  Future<Credentials?> read() async {
    final raw = await _storage.read(key: _key);
    if (raw == null) return null;
    try {
      final m = jsonDecode(raw) as Map<String, dynamic>;
      return Credentials(
        accessToken: m['accessToken'] as String,
        expiresAt: DateTime.parse(m['expiresAt'] as String),
        guardianName: m['guardianName'] as String,
        elders: [
          for (final e in m['elders'] as List<dynamic>)
            ElderSummary(
              id: (e as Map<String, dynamic>)['id'] as String,
              name: e['name'] as String,
            ),
        ],
      );
    } on Object {
      // 형식이 바뀐 옛 값은 버리고 다시 로그인합니다.
      await clear();
      return null;
    }
  }

  @override
  Future<void> write(Credentials c) => _storage.write(
    key: _key,
    value: jsonEncode({
      'accessToken': c.accessToken,
      'expiresAt': c.expiresAt.toUtc().toIso8601String(),
      'guardianName': c.guardianName,
      'elders': [
        for (final e in c.elders) {'id': e.id, 'name': e.name},
      ],
    }),
  );

  @override
  Future<void> clear() => _storage.delete(key: _key);
}

/// 테스트와 개발용 메모리 보관소.
class MemoryCredentialStore implements CredentialStore {
  MemoryCredentialStore([this.value]);

  Credentials? value;

  @override
  Future<Credentials?> read() async => value;

  @override
  Future<void> write(Credentials c) async => value = c;

  @override
  Future<void> clear() async => value = null;
}
