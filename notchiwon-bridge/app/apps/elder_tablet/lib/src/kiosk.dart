import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

/// 키오스크 모드. 어르신이 홈 버튼이나 다른 앱으로 빠져나가지 못하게
/// 화면을 이 앱에 고정합니다 (architecture.md 6.4절).
///
/// 안드로이드 Lock Task는 기기 소유자(Device Owner)로 등록돼 있으면 어르신이
/// 풀 수 없게 잠기고, 아니면 "화면 고정" 안내가 한 번 뜹니다. 네이티브 쪽은
/// `android/app/src/main/kotlin/kr/notchiwon/elder_tablet/MainActivity.kt`,
/// 서비스는 같은 폴더의 `SessionService.kt`입니다.
class Kiosk {
  Kiosk({MethodChannel? channel})
    : _channel = channel ?? const MethodChannel('kr.notchiwon/kiosk');

  final MethodChannel _channel;

  /// 화면을 고정합니다. 기기가 지원하지 않으면 false입니다.
  Future<bool> lock() async => await _call<bool>('lock') ?? false;

  /// 고정을 풉니다 (설정 화면에 들어갈 때).
  Future<void> unlock() => _call<void>('unlock');

  /// 지금 고정돼 있는지.
  Future<bool> isLocked() async => await _call<bool>('isLocked') ?? false;

  /// 화면이 꺼져 있어도 대화가 끊기지 않게 Foreground Service를 켭니다.
  Future<void> startService() => _call<void>('startService');

  Future<void> stopService() => _call<void>('stopService');

  /// 안드로이드가 아니거나(테스트, 데스크톱) 실패하면 null입니다.
  Future<T?> _call<T>(String method) async {
    try {
      return await _channel.invokeMethod<T>(method);
    } on MissingPluginException {
      return null;
    } on PlatformException catch (e) {
      debugPrint('키오스크 $method 실패: ${e.message}');
      return null;
    }
  }
}
