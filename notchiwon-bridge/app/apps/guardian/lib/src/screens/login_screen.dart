import 'package:api_client/api_client.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers.dart';

/// 센터에서 받은 아이디와 비밀번호로 로그인합니다.
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _id = TextEditingController();
  final _password = TextEditingController();
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _id.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_id.text.trim().isEmpty || _password.text.isEmpty) {
      setState(() => _error = '아이디와 비밀번호를 넣어 주세요');
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await ref
          .read(authProvider.notifier)
          .login(_id.text.trim(), _password.text);
    } on ApiException catch (e) {
      setState(
        () => _error = e.code == 401
            ? '아이디나 비밀번호가 맞지 않아요'
            : '로그인하지 못했어요 (${e.code})',
      );
    } on Object {
      setState(() => _error = '서버에 연결하지 못했어요');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    body: SafeArea(
      child: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          const SizedBox(height: 48),
          Text(
            '노치원 보호자',
            style: Theme.of(context).textTheme.headlineMedium,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 32),
          TextField(
            controller: _id,
            decoration: const InputDecoration(labelText: '아이디'),
            textInputAction: TextInputAction.next,
            autocorrect: false,
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _password,
            decoration: const InputDecoration(labelText: '비밀번호'),
            obscureText: true,
            onSubmitted: (_) => _submit(),
          ),
          if (_error != null) ...[
            const SizedBox(height: 12),
            Text(
              _error!,
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
          ],
          const SizedBox(height: 24),
          FilledButton(
            onPressed: _busy ? null : _submit,
            child: Text(_busy ? '로그인 중…' : '로그인'),
          ),
        ],
      ),
    ),
  );
}
