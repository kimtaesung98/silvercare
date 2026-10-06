import 'package:api_client/api_client.dart';
import 'package:elder_tablet/src/session_controller.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:voice/voice.dart';

import 'fakes.dart';

void main() {
  late FakeChannel channel;
  late FakePlayer player;
  late FakeMicrophone mic;
  late SessionController controller;

  Future<void> settle() =>
      Future<void>.delayed(const Duration(milliseconds: 10));

  /// 어르신이 한마디 하고 입을 다문 것처럼 PCM을 밀어넣습니다.
  Future<void> speak() async {
    for (var i = 0; i < 6; i++) {
      mic.push(FakeMicrophone.tone(0.3));
    }
    for (var i = 0; i < 12; i++) {
      mic.push(FakeMicrophone.tone(0.0));
    }
    await settle();
  }

  SessionController newController({bool muteWhileSpeaking = true}) =>
      SessionController(
        socket: ElderSocket(
          serverUrl: Uri.parse('http://server'),
          token: 't',
          connect: (_, _) async => channel,
        ),
        queue: PlaybackQueue(player),
        clips: ClipCache(
          source: FakeClipSource(texts: {'clip-recall-1': '그러셨어요?'}),
          store: MemoryClipStore(),
        ),
        recorder: UtteranceRecorder(mic),
        loadInfo: () async =>
            const TabletInfo(elderName: '김영자', companionEnabled: true),
        muteWhileSpeaking: muteWhileSpeaking,
      );

  setUp(() async {
    channel = FakeChannel();
    player = FakePlayer();
    mic = FakeMicrophone();
    controller = newController();
    await controller.start();
    await settle();
  });

  tearDown(() => controller.dispose());

  /// 연결 직후 서버가 보내는 첫 이벤트 (0번 문장 캐시도 여기서 맞춥니다).
  Future<void> ready() async {
    channel.serverSendsEvent('connection.ready', {
      'deviceId': 'd-1',
      'elderId': 'e-1',
      'openerVersion': 'v1',
    });
    await settle();
  }

  Future<void> startSession({int? eta}) async {
    channel.serverSendsEvent('session.started', {
      'sessionId': 's-1',
      'mode': 'PICKUP_BRIDGE',
      'startedBy': 'CAREGIVER',
      'caregiverEtaMinutes': eta,
    });
    await settle();
  }

  test('연결 직후에는 대기 화면이고, 어르신 호칭을 받아둔다', () async {
    await ready();

    expect(controller.screen, ScreenState.idle);
    expect(controller.connected, isTrue);
    expect(controller.elderName, '김영자 어르신');
  });

  test('세션이 시작되면 대화 화면으로 바뀌고 마이크를 연다', () async {
    await startSession(eta: 12);

    expect(controller.screen, ScreenState.talking);
    expect(controller.etaMinutes, 12);
    expect(mic.started, 1);
  });

  test('조무사가 2분 안으로 들어오면 도착 임박 화면이다', () async {
    await startSession(eta: 12);
    channel.serverSendsEvent('caregiver.eta', {
      'sessionId': 's-1',
      'visitId': 'v-1',
      'minutes': 2,
    });
    await settle();

    expect(controller.screen, ScreenState.arriving);
  });

  test('말동무 모드에서는 ETA가 없어 대화 화면에 머문다', () async {
    channel.serverSendsEvent('session.started', {
      'sessionId': 's-2',
      'mode': 'COMPANION',
      'startedBy': 'ELDER',
    });
    await settle();

    expect(controller.screen, ScreenState.talking);
  });

  test('말동무 요청을 거절하면 어르신 말로 안내한다', () async {
    channel.serverSendsEvent('session.rejected', {'reason': 'BEDTIME'});
    await settle();

    expect(controller.notice, contains('주무실 시간'));
    expect(controller.screen, ScreenState.idle);
  });

  test('0번 문장은 캐시된 음성으로, 1번부터는 서버 MP3로 재생한다', () async {
    await ready();
    await startSession();
    channel.serverSendsEvent('ai.opener', {
      'sessionId': 's-1',
      'turnId': 1,
      'clipId': 'clip-recall-1',
      'category': 'RECALL',
    });
    channel.serverSendsEvent('ai.reply', {
      'sessionId': 's-1',
      'turnId': 1,
      'index': 1,
      'utteranceId': 'u-1',
      'text': '그 시절이 좋았지요.',
    });
    await settle();

    expect(player.played.map((i) => i.index), [0, 1]);
    expect(player.played.first.clipId, 'clip-recall-1');
    expect(controller.captions.map((c) => c.text), ['그러셨어요?', '그 시절이 좋았지요.']);
  });

  test('발화가 끝나면 PCM을 elder.audio로 보낸다', () async {
    await startSession();
    await speak();

    final sent = channel.lastSent('elder.audio');
    expect(sent, isNotNull);
    expect((sent!['data'] as Map)['sessionId'], 's-1');
    // 텍스트 프레임 바로 뒤에 바이너리 프레임이 옵니다.
    expect(channel.sent.last, isA<List<int>>());
  });

  test('재생 중에 말을 시작하면 끼어들기를 보내고 그 턴을 멈춘다', () async {
    // 에코 제거가 되는 기기: 재생 중에도 마이크를 엽니다.
    controller.dispose();
    channel = FakeChannel();
    controller = newController(muteWhileSpeaking: false);
    await controller.start();
    await settle();
    await startSession();
    player.hold = true;
    channel.serverSendsEvent('ai.opener', {
      'sessionId': 's-1',
      'turnId': 3,
      'clipId': 'clip-recall-1',
      'category': 'RECALL',
    });
    await settle();
    mic.push(FakeMicrophone.tone(0.3));
    await settle();

    final sent = channel.lastSent('elder.barge_in');
    expect(sent, isNotNull);
    expect((sent!['data'] as Map)['turnId'], 3);
    expect((sent['data'] as Map)['playedIndex'], 0);
  });

  test('턴이 끝나면 측정값을 보낸다', () async {
    await startSession();
    await speak();
    channel.serverSendsEvent('ai.opener', {
      'sessionId': 's-1',
      'turnId': 4,
      'clipId': 'clip-recall-1',
      'category': 'RECALL',
    });
    await settle();
    channel.serverSendsEvent('ai.turn_end', {
      'sessionId': 's-1',
      'turnId': 4,
      'outcome': 'COMPLETED',
      'sentences': 2,
    });
    await settle();

    final sent = channel.lastSent('client.metrics');
    expect(sent, isNotNull);
    final data = (sent!['data'] as Map).cast<String, dynamic>();
    expect(data['turnId'], 4);
    expect(data['utteranceEndToOpenerMs'], isA<int>());
  });

  test('STT가 실패하면 다시 말해 달라고 안내한다', () async {
    await startSession();
    channel.serverSendsEvent('error', {
      'code': 'STT_FAILED',
      'message': 'clova 502',
    });
    await settle();

    expect(controller.notice, contains('한 번만 더 말씀'));
  });

  test('세션이 끝나면 대기 화면으로 돌아가고 마이크를 닫는다', () async {
    await startSession();
    channel.serverSendsEvent('session.ended', {
      'sessionId': 's-1',
      'reason': 'ELDER_ENDED',
    });
    await settle();

    expect(controller.screen, ScreenState.idle);
    expect(mic.stopped, 1);
  });
}
