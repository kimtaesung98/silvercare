import 'package:flutter/material.dart';

void main() {
  runApp(const ElderTabletApp());
}

/// 어르신 태블릿 앱. 대기 화면과 대화 화면은 단계 4에서 붙입니다.
class ElderTabletApp extends StatelessWidget {
  const ElderTabletApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: '노치원 말동무',
      home: const Scaffold(
        body: Center(child: Text('노치원 말동무', style: TextStyle(fontSize: 48))),
      ),
    );
  }
}
