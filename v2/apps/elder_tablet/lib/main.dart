import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:wakelock_plus/wakelock_plus.dart';

import 'src/chat_client.dart';
import 'src/elder_controller.dart';
import 'src/elder_screen.dart';
import 'src/guardian_screen.dart';
import 'src/voice.dart';

/// v2 서버 주소. 에뮬레이터에서 10.0.2.2는 개발 컴퓨터입니다.
/// 실기기: flutter run --dart-define=SERVER_URL=http://<컴퓨터 IP>:8080
const serverUrl = String.fromEnvironment('SERVER_URL', defaultValue: 'http://10.0.2.2:8080');

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  SystemChrome.setPreferredOrientations([DeviceOrientation.landscapeLeft, DeviceOrientation.landscapeRight]);
  runApp(ElderTabletApp(chat: HttpChat(serverUrl), voice: DeviceVoice(), charger: DeviceCharger()));
}

/// 태블릿 앱: 보호자 화면에서 켜 두면 어르신 화면으로, 시계를 3초 누르면 다시 보호자 화면으로.
class ElderTabletApp extends StatefulWidget {
  const ElderTabletApp({super.key, required this.chat, required this.voice, required this.charger, this.kiosk = true});

  final Chat chat;
  final Voice voice;
  final Charger charger;

  /// 화면 꺼짐 막기와 전체 화면. 시험에서는 끕니다.
  final bool kiosk;

  @override
  State<ElderTabletApp> createState() => _ElderTabletAppState();
}

class _ElderTabletAppState extends State<ElderTabletApp> {
  ElderController? _elder;
  String _name = '';

  void _keepOn(String name) {
    final c = ElderController(chat: widget.chat, voice: widget.voice, elderName: name);
    setState(() {
      _name = name;
      _elder = c;
    });
    if (widget.kiosk) {
      WakelockPlus.enable();
      SystemChrome.setEnabledSystemUIMode(SystemUiMode.immersiveSticky);
    }
    c.start();
  }

  Future<void> _backToGuardian() async {
    final c = _elder;
    setState(() => _elder = null);
    if (widget.kiosk) {
      WakelockPlus.disable();
      SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
    }
    await c?.stop();
    c?.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final elder = _elder;
    return MaterialApp(
      title: '실버케어',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(useMaterial3: true),
      home: elder != null
          ? ElderScreen(controller: elder, onGuardian: _backToGuardian)
          : GuardianScreen(
              chat: widget.chat,
              voice: widget.voice,
              charger: widget.charger,
              initialName: _name,
              onKeepOn: _keepOn,
            ),
    );
  }
}
