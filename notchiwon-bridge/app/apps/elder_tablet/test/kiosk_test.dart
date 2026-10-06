import 'package:elder_tablet/src/kiosk.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('kr.notchiwon/kiosk');
  final calls = <String>[];

  void answer(Object? Function(String method) handler) {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          calls.add(call.method);
          return handler(call.method);
        });
  }

  setUp(calls.clear);

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('화면 고정과 서비스를 네이티브에 넘긴다', () async {
    answer((method) => method == 'lock' ? true : null);

    expect(await Kiosk().lock(), isTrue);
    await Kiosk().startService();

    expect(calls, ['lock', 'startService']);
  });

  test('고정을 못 하는 기기에서는 false를 돌려주고 앱은 그대로 돈다', () async {
    answer((_) => throw PlatformException(code: 'UNSUPPORTED'));

    expect(await Kiosk().lock(), isFalse);
    expect(await Kiosk().isLocked(), isFalse);
  });

  test('안드로이드가 아니면(플러그인 없음) 조용히 넘어간다', () async {
    // 핸들러를 걸지 않으면 MissingPluginException이 납니다.
    expect(await Kiosk().lock(), isFalse);
    await Kiosk().startService();
  });
}
