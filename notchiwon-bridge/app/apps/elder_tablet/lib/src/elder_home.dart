import 'package:flutter/material.dart';
import 'package:ui/ui.dart';

import 'screens/arriving_screen.dart';
import 'screens/idle_screen.dart';
import 'screens/talking_screen.dart';
import 'session_controller.dart';

/// [SessionController]의 상태를 보고 화면 세 개 중 하나를 그립니다.
class ElderHome extends StatelessWidget {
  const ElderHome({required this.controller, super.key});

  final SessionController controller;

  @override
  Widget build(BuildContext context) => ListenableBuilder(
    listenable: controller,
    builder: (context, _) {
      final notice = controller.notice;
      return Scaffold(
        body: SafeArea(
          child: Column(
            children: [
              if (notice != null) NoticeBar(text: notice),
              Expanded(child: _screen()),
            ],
          ),
        ),
      );
    },
  );

  Widget _screen() => switch (controller.screen) {
    ScreenState.idle => IdleScreen(
      elderName: controller.elderName,
      connected: controller.connected,
      onCompanion: controller.companionEnabled
          ? controller.requestCompanion
          : null,
    ),
    ScreenState.talking => TalkingScreen(
      captions: controller.captions,
      speaking: controller.speaking,
      etaMinutes: controller.etaMinutes,
      onEnd: controller.endSession,
    ),
    ScreenState.arriving => ArrivingScreen(
      minutes: controller.etaMinutes ?? 0,
      captions: controller.captions,
      onEnd: controller.endSession,
    ),
  };
}
