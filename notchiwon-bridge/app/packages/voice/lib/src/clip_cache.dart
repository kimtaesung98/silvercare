import 'dart:typed_data';

/// 0번 문장 음성 파일을 어디에 둘지. 실제 구현은 기기 저장소를 쓰고,
/// 테스트는 메모리를 씁니다.
abstract class ClipStore {
  Future<Uint8List?> read(String clipId);

  Future<void> write(String clipId, Uint8List bytes);

  /// 캐시에 있는 clip id 전부.
  Future<Set<String>> ids();

  Future<void> delete(String clipId);

  /// 마지막으로 받아둔 목록 버전.
  Future<String?> readVersion();

  Future<void> writeVersion(String version);
}

/// 서버에서 0번 문장 목록과 음성을 가져오는 통로.
abstract class ClipSource {
  /// `GET /tablet/opener-clips`.
  Future<ClipManifest> manifest();

  /// `GET /tablet/opener-clips/{clipId}/audio`. 아직 음성이 없으면 null.
  Future<Uint8List?> audio(String clipId);
}

/// 0번 문장 목록.
class ClipManifest {
  const ClipManifest({
    required this.version,
    required this.voice,
    required this.clips,
  });

  final String version;
  final String voice;
  final List<ClipInfo> clips;
}

/// 0번 문장 하나.
class ClipInfo {
  const ClipInfo({
    required this.id,
    required this.category,
    required this.text,
    required this.durationMs,
  });

  final String id;
  final String category;
  final String text;
  final int durationMs;
}

/// 0번 문장 음성을 기기에 캐시해 둡니다. 어르신이 말을 마치면 바로 틀어야
/// 해서, 그때 내려받을 시간이 없습니다 (architecture.md 6.2절).
class ClipCache {
  ClipCache({required this.source, required this.store});

  /// 서버에서 목록과 음성을 가져오는 통로.
  final ClipSource source;

  /// 음성을 둘 곳.
  final ClipStore store;
  final _texts = <String, String>{};

  /// 캐시에 있는 음성.
  Future<Uint8List?> audio(String clipId) => store.read(clipId);

  /// 글자 (음성이 없을 때 화면에 띄웁니다).
  String? text(String clipId) => _texts[clipId];

  /// 서버 목록과 맞춥니다. [serverVersion]이 캐시와 같으면 아무것도 받지
  /// 않습니다 (`connection.ready.openerVersion`을 넘기세요).
  /// 받은 음성 개수를 돌려줍니다.
  Future<int> sync({String? serverVersion}) async {
    final cached = await store.readVersion();
    if (serverVersion != null && serverVersion == cached) {
      await _loadTexts();
      return 0;
    }
    final manifest = await source.manifest();
    for (final clip in manifest.clips) {
      _texts[clip.id] = clip.text;
    }

    final have = await store.ids();
    final wanted = {for (final c in manifest.clips) c.id};
    var fetched = 0;
    for (final clip in manifest.clips) {
      if (have.contains(clip.id)) continue;
      final bytes = await source.audio(clip.id);
      if (bytes == null) continue; // 아직 합성 전: 다음 동기화에서 다시 받습니다.
      await store.write(clip.id, bytes);
      fetched++;
    }
    for (final id in have.difference(wanted)) {
      await store.delete(id);
    }
    // 전부 받았을 때만 버전을 적습니다. 그래야 다음에 빠진 음성을 다시 받습니다.
    if (fetched + have.intersection(wanted).length == wanted.length) {
      await store.writeVersion(manifest.version);
    }
    return fetched;
  }

  Future<void> _loadTexts() async {
    if (_texts.isNotEmpty) return;
    final manifest = await source.manifest();
    for (final clip in manifest.clips) {
      _texts[clip.id] = clip.text;
    }
  }
}
