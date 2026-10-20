import 'dart:async';

import 'package:flutter/material.dart';

import 'elder_controller.dart';
import 'guardian_door.dart';
import 'mic_indicator.dart';
import 'tokens.dart';

/// 어르신 화면: 가운데 마이크 원 하나와 상태 글. 어르신은 아무것도 누르지 않습니다.
/// 피그마 프레임 11:37(기다림) · 11:47(듣는 중) · 11:59(말하는 중).
class ElderScreen extends StatelessWidget {
  const ElderScreen({super.key, required this.controller, required this.onGuardian});

  final ElderController controller;
  final VoidCallback onGuardian;

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: false, // 뒤로 가기로 나가지 않게
      child: Scaffold(
        backgroundColor: Tokens.background,
        body: FigmaFrame(
          child: ListenableBuilder(
            listenable: controller,
            builder: (context, _) {
              final (title, help) = _texts(controller.state, controller.offline);
              return Stack(
                children: [
                  Positioned(
                    top: 40,
                    right: 64,
                    child: GuardianDoor(onOpen: onGuardian, child: const _Clock()),
                  ),
                  Positioned(
                    left: 0,
                    right: 0,
                    top: 113,
                    child: Column(
                      children: [
                        MicIndicator(state: controller.offline ? MicState.idle : controller.state),
                        const SizedBox(height: 24),
                        Text(
                          title,
                          style: const TextStyle(fontSize: 52, fontWeight: FontWeight.w700, color: Tokens.text),
                        ),
                        const SizedBox(height: 20),
                        if (help != null) Text(help, style: const TextStyle(fontSize: 32, color: Tokens.textDim)),
                      ],
                    ),
                  ),
                ],
              );
            },
          ),
        ),
      ),
    );
  }

  static (String, String?) _texts(MicState s, bool offline) {
    if (offline) return ('잠시만요', '다시 이어 볼게요');
    return switch (s) {
      MicState.idle => ('말을 걸어 주세요', '"말동무야" 하고 부르시면 대답해요'),
      MicState.listening => ('듣고 있어요', '천천히 말씀하세요'),
      MicState.speaking => ('말동무가 대답해요', null),
    };
  }
}

/// 오른쪽 위 작은 시계 ("오후 3:20").
class _Clock extends StatefulWidget {
  const _Clock();

  @override
  State<_Clock> createState() => _ClockState();
}

class _ClockState extends State<_Clock> {
  late Timer _timer;
  DateTime _now = DateTime.now();

  @override
  void initState() {
    super.initState();
    _timer = Timer.periodic(const Duration(seconds: 15), (_) => setState(() => _now = DateTime.now()));
  }

  @override
  void dispose() {
    _timer.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(8),
      child: Text(
        clockText(_now),
        style: const TextStyle(fontSize: 32, fontWeight: FontWeight.w600, color: Tokens.textDim),
      ),
    );
  }
}

/// 15:20 → "오후 3:20"
String clockText(DateTime t) {
  final h = t.hour % 12 == 0 ? 12 : t.hour % 12;
  return '${t.hour < 12 ? '오전' : '오후'} $h:${t.minute.toString().padLeft(2, '0')}';
}
