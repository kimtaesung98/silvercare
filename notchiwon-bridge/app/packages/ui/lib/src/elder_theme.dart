import 'package:flutter/material.dart';

/// 어르신 태블릿의 색과 글자 크기.
///
/// 노안과 백내장을 생각해 밝은 바탕에 검은 글씨, 큰 글자를 씁니다.
/// 파란 바탕에 흰 글씨 같은 조합은 대비가 낮아 쓰지 않습니다.
abstract final class ElderColors {
  /// 바탕 (따뜻한 흰색: 순백보다 눈이 덜 피곤합니다).
  static const background = Color(0xFFFAF7F2);

  /// 글자.
  static const text = Color(0xFF1A1A1A);

  /// 어르신 말(자막) 배경.
  static const elderBubble = Color(0xFFE8EEF7);

  /// AI 말(자막) 배경.
  static const aiBubble = Color(0xFFFFF1DC);

  /// 말동무 버튼.
  static const action = Color(0xFF1B5E9A);

  /// 그만하기 버튼.
  static const quiet = Color(0xFF6B6B6B);

  /// 도착 임박 화면.
  static const arriving = Color(0xFF1E6B3A);

  /// 안내·오류 띠.
  static const notice = Color(0xFFFFE08A);
}

/// 어르신 앱 테마. 글자는 기본 18pt 대신 크게 둡니다.
ThemeData elderTheme() {
  final base = ThemeData.light(useMaterial3: true);
  return base.copyWith(
    scaffoldBackgroundColor: ElderColors.background,
    colorScheme: base.colorScheme.copyWith(
      primary: ElderColors.action,
      surface: ElderColors.background,
    ),
    textTheme: base.textTheme
        .apply(
          bodyColor: ElderColors.text,
          displayColor: ElderColors.text,
          fontSizeFactor: 1.0,
        )
        .copyWith(
          displayLarge: const TextStyle(
            fontSize: 84,
            fontWeight: FontWeight.w700,
          ),
          headlineLarge: const TextStyle(
            fontSize: 56,
            fontWeight: FontWeight.w700,
          ),
          titleLarge: const TextStyle(
            fontSize: 40,
            fontWeight: FontWeight.w600,
          ),
          bodyLarge: const TextStyle(fontSize: 34, height: 1.4),
          bodyMedium: const TextStyle(fontSize: 28, height: 1.4),
        ),
  );
}
