import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

/// v2 서버 채팅 바탕의 메시지 하나 (`chat.Message`).
class ChatMessage {
  ChatMessage({
    required this.seq,
    required this.fromId,
    required this.fromName,
    required this.fromKind,
    required this.text,
    this.tags = const {},
  });

  factory ChatMessage.fromJson(Map<String, dynamic> j) {
    final from = j['from'] as Map<String, dynamic>;
    return ChatMessage(
      seq: j['seq'] as int,
      fromId: from['id'] as String,
      fromName: from['name'] as String,
      fromKind: from['kind'] as String,
      text: j['text'] as String,
      tags: (j['tags'] as Map<String, dynamic>? ?? {}).cast<String, String>(),
    );
  }

  final int seq;
  final String fromId;
  final String fromName;

  /// human · ai · system
  final String fromKind;
  final String text;
  final Map<String, String> tags;
}

/// 방에 들어온 결과: 방 id와 서버가 정해 준 내 참가자 id.
class Joined {
  const Joined(this.roomId, this.meId);
  final String roomId;
  final String meId;
}

/// 태블릿이 서버 채팅에 붙는 길. 시험에서는 가짜로 바꿔 끼웁니다.
abstract class Chat {
  /// 서버가 살아 있는지 봅니다 (보호자 화면의 "인터넷" 확인).
  Future<bool> healthy();

  /// 어르신 이름으로 새 방을 엽니다. 서버가 말동무와 안전 지킴이를 함께 넣습니다.
  Future<Joined> openRoom(String elderName);

  Future<void> post(Joined me, String text);

  /// 방의 메시지 흐름. 지난 메시지가 먼저 오고, 연결이 끊기면 흐름이 끝납니다.
  Stream<ChatMessage> events(String roomId);
}

/// v2 서버(`cmd/server`)의 HTTP API를 씁니다.
class HttpChat implements Chat {
  HttpChat(this.baseUrl, {http.Client? client}) : _client = client ?? http.Client();

  final String baseUrl;
  final http.Client _client;

  Uri _u(String path) => Uri.parse('$baseUrl$path');

  @override
  Future<bool> healthy() async {
    try {
      final r = await _client.get(_u('/api/health')).timeout(const Duration(seconds: 3));
      return r.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  @override
  Future<Joined> openRoom(String elderName) async {
    final j = await _postJson('/api/rooms', {'name': elderName});
    return Joined(j['roomId'] as String, (j['me'] as Map<String, dynamic>)['id'] as String);
  }

  @override
  Future<void> post(Joined me, String text) async {
    await _postJson('/api/rooms/${me.roomId}/messages', {'from': me.meId, 'text': text});
  }

  Future<Map<String, dynamic>> _postJson(String path, Object body) async {
    final r = await _client
        .post(_u(path), headers: {'content-type': 'application/json'}, body: jsonEncode(body))
        .timeout(const Duration(seconds: 10));
    final j = jsonDecode(utf8.decode(r.bodyBytes)) as Map<String, dynamic>;
    if (r.statusCode != 200) throw Exception(j['error'] ?? 'HTTP ${r.statusCode}');
    return j;
  }

  @override
  Stream<ChatMessage> events(String roomId) {
    // 흐름마다 따로 연결을 쓰고, 구독을 끊으면 연결도 닫습니다 (오래 열린 SSE 연결이라).
    final client = http.Client();
    final out = StreamController<ChatMessage>(onCancel: client.close);
    () async {
      try {
        final resp = await client.send(http.Request('GET', _u('/api/rooms/$roomId/events')));
        if (resp.statusCode != 200) throw Exception('events HTTP ${resp.statusCode}');
        final lines = resp.stream.transform(utf8.decoder).transform(const LineSplitter());
        await for (final line in lines) {
          if (line.startsWith('data: ')) {
            out.add(ChatMessage.fromJson(jsonDecode(line.substring(6)) as Map<String, dynamic>));
          }
        }
      } catch (e) {
        if (!out.isClosed) out.addError(e);
      }
      if (!out.isClosed) await out.close();
      client.close();
    }();
    return out.stream;
  }
}
