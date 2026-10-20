import 'package:flutter/material.dart';

/// 피그마 파일의 `실버케어` 변수 모음과 같은 값입니다. 피그마에서 바꾸면 여기도 바꿉니다.
abstract final class Tokens {
  static const background = Color(0xFFFFF8EE); // 색/배경
  static const text = Color(0xFF2B2118); // 색/글자
  static const textDim = Color(0xFF7A6A5C); // 색/글자-흐림
  static const accent = Color(0xFFD46B08); // 색/강조
  static const listening = Color(0xFF389E0D); // 색/듣는중
  static const white = Color(0xFFFFFFFF); // 색/흰색

  /// 피그마 프레임 크기. 화면은 이 크기로 그린 뒤 기기에 맞춰 늘이고 줄입니다.
  static const frame = Size(1280, 800);
}

/// 피그마 프레임 크기(1280×800)로 그린 화면을 기기 화면에 맞춥니다.
class FigmaFrame extends StatelessWidget {
  const FigmaFrame({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: Tokens.background,
      child: Center(
        child: FittedBox(
          child: SizedBox.fromSize(size: Tokens.frame, child: child),
        ),
      ),
    );
  }
}
