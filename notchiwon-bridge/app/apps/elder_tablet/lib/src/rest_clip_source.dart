import 'dart:typed_data';

import 'package:api_client/api_client.dart';
import 'package:http/http.dart' as http;
import 'package:voice/voice.dart';

/// 서버 REST로 0번 문장 목록과 음성을 받아오는 [ClipSource].
///
/// 목록은 생성된 [TabletApi]를 쓰고, 음성은 바이트를 그대로 받아야 해서
/// `http`로 직접 받습니다.
class RestClipSource implements ClipSource {
  RestClipSource({
    required this.serverUrl,
    required this.token,
    http.Client? client,
  }) : _client = client ?? http.Client(),
       _api = TabletApi(
         ApiClient(
           basePath: serverUrl.toString().replaceAll(RegExp(r'/$'), ''),
           authentication: HttpBearerAuth()..accessToken = token,
         ),
       );

  /// 서버 주소 (`http://`나 `https://`).
  final Uri serverUrl;

  /// 기기 토큰.
  final String token;

  final http.Client _client;
  final TabletApi _api;

  @override
  Future<ClipManifest> manifest() async {
    final manifest = await _api.listOpenerClips();
    if (manifest == null) {
      throw StateError('0번 문장 목록이 비어 있습니다');
    }
    return ClipManifest(
      version: manifest.version,
      voice: manifest.voice,
      clips: [
        for (final clip in manifest.clips)
          ClipInfo(
            id: clip.id,
            category: clip.category.toJson(),
            text: clip.text,
            durationMs: clip.durationMs,
          ),
      ],
    );
  }

  @override
  Future<Uint8List?> audio(String clipId) async {
    final response = await _client.get(
      serverUrl.replace(path: '/tablet/opener-clips/$clipId/audio'),
      headers: {'Authorization': 'Bearer $token'},
    );
    // 404는 아직 합성 전입니다 (다음 동기화에서 다시 받습니다).
    if (response.statusCode == 404) return null;
    if (response.statusCode != 200) {
      throw http.ClientException(
        '0번 문장 음성 $clipId: HTTP ${response.statusCode}',
      );
    }
    return response.bodyBytes;
  }

  void close() => _client.close();
}
