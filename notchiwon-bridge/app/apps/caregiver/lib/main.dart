import 'package:flutter/material.dart';

void main() {
  runApp(const CaregiverApp());
}

/// 조무사 앱. 방문 목록과 브리핑 화면은 단계 5에서 붙입니다.
class CaregiverApp extends StatelessWidget {
  const CaregiverApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: '노치원 조무사',
      home: const Scaffold(body: Center(child: Text('노치원 조무사'))),
    );
  }
}
