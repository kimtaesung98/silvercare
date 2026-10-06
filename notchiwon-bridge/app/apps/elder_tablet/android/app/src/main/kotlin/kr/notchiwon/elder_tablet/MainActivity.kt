package kr.notchiwon.elder_tablet

import android.app.ActivityManager
import android.content.Context
import android.content.Intent
import android.os.Build
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

/**
 * 어르신 태블릿의 네이티브 쪽.
 *
 * 키오스크(Lock Task)와 대화용 Foreground Service를 Dart(`lib/src/kiosk.dart`)에
 * 열어 줍니다.
 */
class MainActivity : FlutterActivity() {
    private val channelName = "kr.notchiwon/kiosk"

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, channelName)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "lock" -> result.success(startKiosk())
                    "unlock" -> {
                        stopKiosk()
                        result.success(null)
                    }
                    "isLocked" -> result.success(isLocked())
                    "startService" -> {
                        SessionService.start(this)
                        result.success(null)
                    }
                    "stopService" -> {
                        SessionService.stop(this)
                        result.success(null)
                    }
                    else -> result.notImplemented()
                }
            }
    }

    /**
     * 화면을 이 앱에 고정합니다.
     *
     * 기기 소유자로 등록돼 있으면 어르신이 풀 수 없게 잠기고, 아니면 안드로이드가
     * "화면 고정" 안내를 한 번 띄웁니다. 둘 다 실패하면 false를 돌려줘서
     * 앱이 그냥 보통 화면으로 동작하게 합니다.
     */
    private fun startKiosk(): Boolean =
        try {
            startLockTask()
            true
        } catch (e: IllegalStateException) {
            // 화면 고정을 쓸 수 없는 기기입니다.
            false
        }

    private fun stopKiosk() {
        try {
            if (isLocked()) stopLockTask()
        } catch (e: IllegalStateException) {
            // 이미 풀려 있습니다.
        }
    }

    private fun isLocked(): Boolean {
        val manager = getSystemService(Context.ACTIVITY_SERVICE) as ActivityManager
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            manager.lockTaskModeState != ActivityManager.LOCK_TASK_MODE_NONE
        } else {
            @Suppress("DEPRECATION")
            manager.isInLockTaskMode
        }
    }

    override fun onDestroy() {
        // 앱이 내려가면 서비스도 함께 내립니다.
        stopService(Intent(this, SessionService::class.java))
        super.onDestroy()
    }
}
