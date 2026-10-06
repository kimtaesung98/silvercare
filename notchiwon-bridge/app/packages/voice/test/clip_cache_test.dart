import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:voice/voice.dart';

class MemoryStore implements ClipStore {
  final files = <String, Uint8List>{};
  String? version;

  @override
  Future<Uint8List?> read(String clipId) async => files[clipId];

  @override
  Future<void> write(String clipId, Uint8List bytes) async =>
      files[clipId] = bytes;

  @override
  Future<Set<String>> ids() async => files.keys.toSet();

  @override
  Future<void> delete(String clipId) async => files.remove(clipId);

  @override
  Future<String?> readVersion() async => version;

  @override
  Future<void> writeVersion(String v) async => version = v;
}

class FakeSource implements ClipSource {
  FakeSource(this.version, this.clipIds, {this.missingAudio = const {}});

  String version;
  List<String> clipIds;
  Set<String> missingAudio;
  var manifestCalls = 0;
  final fetched = <String>[];

  @override
  Future<ClipManifest> manifest() async {
    manifestCalls++;
    return ClipManifest(
      version: version,
      voice: 'default',
      clips: [
        for (final id in clipIds)
          ClipInfo(
            id: id,
            category: 'RECALL',
            text: '문장 $id',
            durationMs: 1200,
          ),
      ],
    );
  }

  @override
  Future<Uint8List?> audio(String clipId) async {
    fetched.add(clipId);
    if (missingAudio.contains(clipId)) return null;
    return Uint8List.fromList(clipId.codeUnits);
  }
}

void main() {
  test('처음에는 전부 받고 버전을 적는다', () async {
    final store = MemoryStore();
    final source = FakeSource('v1', ['a', 'b']);
    final cache = ClipCache(source: source, store: store);

    expect(await cache.sync(), 2);
    expect(store.version, 'v1');
    expect(await cache.audio('a'), 'a'.codeUnits);
    expect(cache.text('b'), '문장 b');
  });

  test('버전이 같으면 음성을 다시 받지 않는다', () async {
    final store = MemoryStore()..version = 'v1';
    final source = FakeSource('v1', ['a']);
    final cache = ClipCache(source: source, store: store);

    expect(await cache.sync(serverVersion: 'v1'), 0);
    expect(source.fetched, isEmpty);
    expect(cache.text('a'), '문장 a', reason: '글자는 목록에서 읽어 둔다');
  });

  test('버전이 바뀌면 새 음성만 받고 빠진 클립은 지운다', () async {
    final store = MemoryStore()
      ..version = 'v1'
      ..files['a'] = Uint8List.fromList([1])
      ..files['old'] = Uint8List.fromList([2]);
    final source = FakeSource('v2', ['a', 'b']);
    final cache = ClipCache(source: source, store: store);

    expect(await cache.sync(serverVersion: 'v2'), 1);
    expect(source.fetched, ['b']);
    expect(store.files.keys, unorderedEquals(['a', 'b']));
    expect(store.version, 'v2');
  });

  test('아직 합성되지 않은 음성이 있으면 버전을 적지 않는다', () async {
    final store = MemoryStore();
    final source = FakeSource('v1', ['a', 'b'], missingAudio: {'b'});
    final cache = ClipCache(source: source, store: store);

    expect(await cache.sync(), 1);
    expect(store.version, isNull, reason: '다음에 b를 다시 받아야 한다');

    source.missingAudio = {};
    expect(await cache.sync(), 1);
    expect(store.version, 'v1');
  });
}
