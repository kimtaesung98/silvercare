import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:just_audio/just_audio.dart';
import 'package:path_provider/path_provider.dart';
import 'package:record/record.dart';

import 'clip_cache.dart';
import 'playback_queue.dart';
import 'recorder.dart';

/// 기기에서 실제로 소리를 내는 [AudioPlayerPort].
///
/// 0번 문장은 캐시 폴더의 파일을, 서버가 보낸 문장은 메모리의 MP3를 틉니다.
class JustAudioPlayer implements AudioPlayerPort {
  JustAudioPlayer({required this.store, AudioPlayer? player})
    : _player = player ?? AudioPlayer();

  /// 0번 문장 음성이 있는 곳.
  final FileClipStore store;
  var _replyFiles = 0;
  final AudioPlayer _player;

  @override
  Future<void> play(PlayItem item) async {
    final clipId = item.clipId;
    final bytes = item.audio;
    if (clipId != null) {
      final file = store.fileOf(clipId);
      if (!file.existsSync()) return; // 아직 못 받은 음성: 글자만 보여줍니다.
      await _player.setFilePath(file.path);
    } else if (bytes != null) {
      // just_audio는 파일이나 URL을 틉니다. 문장 하나는 수십 KB라 임시 파일로
      // 쓰는 편이 메모리 소스(실험 API)보다 단순합니다.
      final file = File(
        '${Directory.systemTemp.path}/reply_${_replyFiles++ % 4}.mp3',
      );
      await file.writeAsBytes(bytes, flush: true);
      await _player.setFilePath(file.path);
    } else {
      return;
    }
    await _player.play();
    await _player.processingStateStream.firstWhere(
      (s) => s == ProcessingState.completed,
    );
    await _player.stop();
  }

  @override
  Future<void> stop() => _player.stop();

  Future<void> dispose() => _player.dispose();
}

/// 0번 문장 음성을 앱 지원 폴더에 두는 [ClipStore].
class FileClipStore implements ClipStore {
  FileClipStore(this._dir);

  /// 기기의 앱 지원 폴더 아래 `opener_clips/`를 씁니다.
  static Future<FileClipStore> open() async {
    final base = await getApplicationSupportDirectory();
    final dir = Directory('${base.path}/opener_clips');
    await dir.create(recursive: true);
    return FileClipStore(dir);
  }

  final Directory _dir;

  File fileOf(String clipId) => File('${_dir.path}/$clipId.mp3');

  File get _versionFile => File('${_dir.path}/version.txt');

  @override
  Future<Uint8List?> read(String clipId) async {
    final file = fileOf(clipId);
    return file.existsSync() ? file.readAsBytes() : null;
  }

  @override
  Future<void> write(String clipId, Uint8List bytes) =>
      fileOf(clipId).writeAsBytes(bytes, flush: true);

  @override
  Future<Set<String>> ids() async => {
    for (final f in _dir.listSync())
      if (f is File && f.path.endsWith('.mp3'))
        f.uri.pathSegments.last.replaceAll('.mp3', ''),
  };

  @override
  Future<void> delete(String clipId) async {
    final file = fileOf(clipId);
    if (file.existsSync()) await file.delete();
  }

  @override
  Future<String?> readVersion() async =>
      _versionFile.existsSync() ? _versionFile.readAsString() : null;

  @override
  Future<void> writeVersion(String version) =>
      _versionFile.writeAsString(version, flush: true);
}

/// `record` 패키지로 16kHz 모노 PCM16을 내보내는 마이크.
class DeviceMicrophone implements MicrophonePort {
  DeviceMicrophone({AudioRecorder? recorder})
    : _recorder = recorder ?? AudioRecorder();

  final AudioRecorder _recorder;

  /// 마이크 권한이 있는지 (없으면 물어봅니다).
  Future<bool> hasPermission() => _recorder.hasPermission();

  @override
  Future<Stream<Uint8List>> start() => _recorder.startStream(
    const RecordConfig(
      encoder: AudioEncoder.pcm16bits,
      sampleRate: 16000,
      numChannels: 1,
      // 스피커 소리가 마이크로 되돌아오는 것을 기기가 줄여 줍니다.
      echoCancel: true,
      noiseSuppress: true,
    ),
  );

  @override
  Future<void> stop() async {
    if (await _recorder.isRecording()) await _recorder.stop();
  }

  Future<void> dispose() => _recorder.dispose();
}
