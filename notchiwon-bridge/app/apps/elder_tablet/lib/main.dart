import 'package:api_client/api_client.dart';
import 'package:flutter/material.dart';
import 'package:ui/ui.dart';
import 'package:voice/voice.dart';

import 'src/elder_home.dart';
import 'src/kiosk.dart';
import 'src/rest_clip_source.dart';
import 'src/session_controller.dart';

/// 서버 주소. 빌드할 때 넣습니다:
/// `flutter build apk --dart-define=SERVER_URL=https://...`.
/// 기본값은 안드로이드 에뮬레이터에서 보이는 개발 서버입니다.
const serverUrl = String.fromEnvironment(
  'SERVER_URL',
  defaultValue: 'http://10.0.2.2:8000',
);

/// 기기 토큰. `POST /devices/{deviceId}/token`으로 발급한 값을 넣습니다.
const deviceToken = String.fromEnvironment('DEVICE_TOKEN');

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final url = Uri.parse(serverUrl);
  final socket = ElderSocket(serverUrl: url, token: deviceToken);
  final microphone = DeviceMicrophone();
  final store = await FileClipStore.open();
  final clips = ClipCache(
    source: RestClipSource(serverUrl: url, token: deviceToken),
    store: store,
  );
  final controller = SessionController(
    socket: socket,
    queue: PlaybackQueue(JustAudioPlayer(store: store)),
    clips: clips,
    recorder: UtteranceRecorder(microphone),
    loadInfo: () => _loadInfo(url, deviceToken),
  );

  // 마이크 권한이 없으면 어르신이 답해야 합니다. 태블릿을 설치할 때 한 번만
  // 묻고, 그다음부터는 바로 통과합니다.
  await microphone.hasPermission();

  final kiosk = Kiosk();
  await kiosk.lock();
  await kiosk.startService();

  await controller.start();
  runApp(ElderTabletApp(controller: controller));
}

/// `GET /tablet/context`로 어르신 호칭과 말동무 사용 여부를 받아옵니다.
Future<TabletInfo> _loadInfo(Uri serverUrl, String token) async {
  final api = TabletApi(
    ApiClient(
      basePath: serverUrl.toString().replaceAll(RegExp(r'/$'), ''),
      authentication: HttpBearerAuth()..accessToken = token,
    ),
  );
  final context = await api.getTabletContext();
  if (context == null) throw StateError('기기 정보를 받지 못했습니다');
  return TabletInfo(
    elderName: context.elder.name,
    companionEnabled: context.companionEnabled,
  );
}

/// 어르신 태블릿 앱.
class ElderTabletApp extends StatelessWidget {
  const ElderTabletApp({required this.controller, super.key});

  final SessionController controller;

  @override
  Widget build(BuildContext context) => MaterialApp(
    title: '말동무',
    theme: elderTheme(),
    debugShowCheckedModeBanner: false,
    home: ElderHome(controller: controller),
  );
}
