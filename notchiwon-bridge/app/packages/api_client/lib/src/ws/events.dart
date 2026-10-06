/// `/ws/elder`에서 오가는 이벤트. 서버 계약은 `server/api/ws-events.md`이고
/// 형태의 기준은 `ws-events.schema.json`입니다.
///
/// 이 파일은 손으로 관리합니다. 서버가 보내는 이벤트는 [ElderEvent.decode]로
/// 읽고, 태블릿이 보내는 이벤트는 [ElderCommand]로 만듭니다.
library;

import 'dart:convert';
import 'dart:typed_data';

import '../generated/api.dart';

// 0번 문장 분류(OpenerCategory)와 대화 모드(SessionMode)는 REST 계약과 같은
// 값이라 생성 코드의 enum을 그대로 씁니다.

/// 서버가 모르는 값을 보내도 멈추지 않도록, 못 읽으면 회상으로 둡니다.
OpenerCategory parseOpenerCategory(Object? value) =>
    OpenerCategory.fromJson(value) ?? OpenerCategory.RECALL;

/// 못 읽으면 말동무로 둡니다.
SessionMode parseSessionMode(Object? value) =>
    SessionMode.fromJson(value) ?? SessionMode.COMPANION;

/// 한 턴이 끝난 이유 (`ai.turn_end.outcome`).
enum TurnOutcome {
  completed('COMPLETED'),
  fallback('FALLBACK'),
  escalated('ESCALATED'),
  cancelled('CANCELLED');

  const TurnOutcome(this.wire);

  final String wire;

  static TurnOutcome parse(Object? value) => TurnOutcome.values.firstWhere(
    (o) => o.wire == value,
    orElse: () => TurnOutcome.completed,
  );
}

/// 서버가 보내는 이벤트.
sealed class ElderEvent {
  const ElderEvent();

  /// 텍스트 프레임 하나를 읽습니다. 모르는 `type`이면 [UnknownEvent]입니다.
  /// JSON이 아니거나 봉투 모양이 아니면 [FormatException]을 던집니다.
  static ElderEvent decode(String frame) {
    final Object? parsed = jsonDecode(frame);
    if (parsed is! Map<String, dynamic>) {
      throw const FormatException('ws 프레임이 객체가 아닙니다');
    }
    final type = parsed['type'];
    final data = parsed['data'];
    if (type is! String || data is! Map<String, dynamic>) {
      throw const FormatException('ws 봉투에 type이나 data가 없습니다');
    }
    return switch (type) {
      'connection.ready' => ConnectionReady.fromData(data),
      'session.started' => SessionStarted.fromData(data),
      'session.rejected' => SessionRejected(reason: data['reason'] as String),
      'session.ended' => SessionEnded(
        sessionId: data['sessionId'] as String,
        reason: data['reason'] as String,
      ),
      'elder.transcript' => ElderTranscript.fromData(data),
      'ai.opener' => AiOpener.fromData(data),
      'ai.filler' => AiFiller(
        sessionId: data['sessionId'] as String,
        turnId: data['turnId'] as int,
        clipId: data['clipId'] as String,
      ),
      'ai.reply' => AiReply.fromData(data),
      'ai.turn_end' => AiTurnEnd.fromData(data),
      'caregiver.eta' => CaregiverEta(
        sessionId: data['sessionId'] as String,
        visitId: data['visitId'] as String,
        minutes: data['minutes'] as int,
      ),
      'error' => ServerError(
        code: data['code'] as String,
        message: data['message'] as String,
        replyTo: data['replyTo'] as String?,
      ),
      'ping' => Ping(nonce: data['nonce'] as String),
      'pong' => Pong(nonce: data['nonce'] as String),
      _ => UnknownEvent(type: type, data: data),
    };
  }

  /// 이 이벤트 바로 다음에 오디오 바이너리 프레임이 오면 그 길이, 아니면 null.
  int? get expectedAudioBytes => null;

  /// 뒤따라온 오디오를 붙인 이벤트를 돌려줍니다.
  ElderEvent withAudio(Uint8List audio) => this;
}

/// 재연결했을 때 이어갈 세션.
class ResumedSession {
  const ResumedSession({required this.sessionId, required this.mode});

  final String sessionId;
  final SessionMode mode;
}

/// 연결 직후 한 번.
class ConnectionReady extends ElderEvent {
  const ConnectionReady({
    required this.deviceId,
    required this.elderId,
    required this.openerVersion,
    this.activeSession,
  });

  factory ConnectionReady.fromData(Map<String, dynamic> data) {
    final active = data['activeSession'];
    return ConnectionReady(
      deviceId: data['deviceId'] as String,
      elderId: data['elderId'] as String,
      openerVersion: data['openerVersion'] as String,
      activeSession: active is Map<String, dynamic>
          ? ResumedSession(
              sessionId: active['sessionId'] as String,
              mode: parseSessionMode(active['mode']),
            )
          : null,
    );
  }

  final String deviceId;
  final String elderId;

  /// `GET /tablet/opener-clips`의 version. 캐시와 다르면 다시 내려받습니다.
  final String openerVersion;
  final ResumedSession? activeSession;
}

/// 세션 시작.
class SessionStarted extends ElderEvent {
  const SessionStarted({
    required this.sessionId,
    required this.mode,
    required this.startedBy,
    this.caregiverEtaMinutes,
  });

  factory SessionStarted.fromData(Map<String, dynamic> data) => SessionStarted(
    sessionId: data['sessionId'] as String,
    mode: parseSessionMode(data['mode']),
    startedBy: data['startedBy'] as String,
    caregiverEtaMinutes: data['caregiverEtaMinutes'] as int?,
  );

  final String sessionId;
  final SessionMode mode;
  final String startedBy;
  final int? caregiverEtaMinutes;
}

/// 말동무 요청 거절.
class SessionRejected extends ElderEvent {
  const SessionRejected({required this.reason});

  final String reason;
}

/// 세션 종료.
class SessionEnded extends ElderEvent {
  const SessionEnded({required this.sessionId, required this.reason});

  final String sessionId;
  final String reason;
}

/// 어르신 발화의 음성 인식 결과.
class ElderTranscript extends ElderEvent {
  const ElderTranscript({
    required this.sessionId,
    required this.seq,
    required this.utteranceId,
    required this.text,
    this.clientId,
  });

  factory ElderTranscript.fromData(Map<String, dynamic> data) =>
      ElderTranscript(
        sessionId: data['sessionId'] as String,
        seq: data['seq'] as int,
        utteranceId: data['utteranceId'] as String,
        text: data['text'] as String,
        clientId: data['clientId'] as String?,
      );

  final String sessionId;
  final int seq;
  final String utteranceId;
  final String text;
  final String? clientId;
}

/// 0번 문장. 태블릿은 캐시된 음성을 바로 재생합니다.
class AiOpener extends ElderEvent {
  const AiOpener({
    required this.sessionId,
    required this.turnId,
    required this.clipId,
    required this.category,
  });

  factory AiOpener.fromData(Map<String, dynamic> data) => AiOpener(
    sessionId: data['sessionId'] as String,
    turnId: data['turnId'] as int,
    clipId: data['clipId'] as String,
    category: parseOpenerCategory(data['category']),
  );

  final String sessionId;
  final int turnId;
  final String clipId;
  final OpenerCategory category;
}

/// 1번 문장이 늦을 때 트는 추임새.
class AiFiller extends ElderEvent {
  const AiFiller({
    required this.sessionId,
    required this.turnId,
    required this.clipId,
  });

  final String sessionId;
  final int turnId;
  final String clipId;
}

/// 스트리밍 문장 하나 (1번부터). [audio]는 서버에 음성 합성이 없으면 null입니다.
class AiReply extends ElderEvent {
  const AiReply({
    required this.sessionId,
    required this.turnId,
    required this.index,
    required this.utteranceId,
    required this.text,
    this.audioBytes,
    this.audio,
  });

  factory AiReply.fromData(Map<String, dynamic> data) {
    final audio = data['audio'];
    return AiReply(
      sessionId: data['sessionId'] as String,
      turnId: data['turnId'] as int,
      index: data['index'] as int,
      utteranceId: data['utteranceId'] as String,
      text: data['text'] as String,
      audioBytes: audio is Map<String, dynamic> ? audio['bytes'] as int : null,
    );
  }

  final String sessionId;
  final int turnId;

  /// 턴 안의 문장 번호. 0번은 [AiOpener]이므로 1부터입니다.
  final int index;
  final String utteranceId;
  final String text;

  /// 뒤따라올 오디오의 길이 (없으면 null).
  final int? audioBytes;

  /// 뒤따라온 MP3.
  final Uint8List? audio;

  @override
  int? get expectedAudioBytes => audioBytes;

  @override
  AiReply withAudio(Uint8List audio) => AiReply(
    sessionId: sessionId,
    turnId: turnId,
    index: index,
    utteranceId: utteranceId,
    text: text,
    audioBytes: audioBytes,
    audio: audio,
  );
}

/// 이 턴에 더 올 문장이 없음.
class AiTurnEnd extends ElderEvent {
  const AiTurnEnd({
    required this.sessionId,
    required this.turnId,
    required this.outcome,
    required this.sentences,
  });

  factory AiTurnEnd.fromData(Map<String, dynamic> data) => AiTurnEnd(
    sessionId: data['sessionId'] as String,
    turnId: data['turnId'] as int,
    outcome: TurnOutcome.parse(data['outcome']),
    sentences: data['sentences'] as int,
  );

  final String sessionId;
  final int turnId;
  final TurnOutcome outcome;
  final int sentences;
}

/// 조무사 도착까지 남은 분.
class CaregiverEta extends ElderEvent {
  const CaregiverEta({
    required this.sessionId,
    required this.visitId,
    required this.minutes,
  });

  final String sessionId;
  final String visitId;
  final int minutes;
}

/// 서버가 메시지를 처리하지 못함 (연결은 유지).
class ServerError extends ElderEvent {
  const ServerError({required this.code, required this.message, this.replyTo});

  final String code;
  final String message;
  final String? replyTo;
}

/// 연결 확인.
class Ping extends ElderEvent {
  const Ping({required this.nonce});

  final String nonce;
}

/// [Ping]의 답.
class Pong extends ElderEvent {
  const Pong({required this.nonce});

  final String nonce;
}

/// 이 앱이 모르는 이벤트 (서버가 새 이벤트를 늘린 경우).
class UnknownEvent extends ElderEvent {
  const UnknownEvent({required this.type, required this.data});

  final String type;
  final Map<String, dynamic> data;
}

/// 태블릿이 서버로 보내는 메시지.
class ElderCommand {
  const ElderCommand(this.type, this.data, {this.audio});

  /// 말동무 버튼.
  factory ElderCommand.requestCompanion() =>
      const ElderCommand('session.request', {'mode': 'COMPANION'});

  /// 그만하기 버튼.
  factory ElderCommand.endSession(String sessionId) =>
      ElderCommand('session.end', {'sessionId': sessionId});

  /// 발화 한 구간. [audio]가 바로 다음 바이너리 프레임으로 나갑니다.
  factory ElderCommand.audio({
    required String sessionId,
    required String clientId,
    required Uint8List audio,
    String format = 'pcm16',
    int sampleRateHz = 16000,
  }) => ElderCommand('elder.audio', {
    'sessionId': sessionId,
    'clientId': clientId,
    'audio': {
      'format': format,
      'sampleRateHz': sampleRateHz,
      'bytes': audio.length,
    },
  }, audio: audio);

  /// 텍스트 모드 발화 (개발용).
  factory ElderCommand.text({
    required String sessionId,
    required String clientId,
    required String text,
  }) => ElderCommand('elder.text', {
    'sessionId': sessionId,
    'clientId': clientId,
    'text': text,
  });

  /// 재생 중 어르신이 말을 시작함.
  factory ElderCommand.bargeIn({
    required String sessionId,
    required int turnId,
    required int playedIndex,
  }) => ElderCommand('elder.barge_in', {
    'sessionId': sessionId,
    'turnId': turnId,
    'playedIndex': playedIndex,
  });

  /// 한 턴의 체감 지연 (단계 4 측정).
  factory ElderCommand.metrics({
    required String sessionId,
    required int turnId,
    int? utteranceEndToOpenerMs,
    int? openerEndToFirstReplyMs,
    int? utteranceEndToFirstReplyMs,
  }) => ElderCommand('client.metrics', {
    'sessionId': sessionId,
    'turnId': turnId,
    'utteranceEndToOpenerMs': utteranceEndToOpenerMs,
    'openerEndToFirstReplyMs': openerEndToFirstReplyMs,
    'utteranceEndToFirstReplyMs': utteranceEndToFirstReplyMs,
  });

  factory ElderCommand.pong(String nonce) =>
      ElderCommand('pong', {'nonce': nonce});

  final String type;
  final Map<String, dynamic> data;
  final Uint8List? audio;

  /// 보낼 텍스트 프레임.
  String encode({String? id}) => jsonEncode({
    'type': type,
    'id': ?id,
    'ts': DateTime.now().toUtc().toIso8601String(),
    'data': data,
  });
}
