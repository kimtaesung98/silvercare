package kr.notchiwon.guardian

import android.app.NotificationChannel
import android.app.NotificationManager
import android.media.AudioAttributes
import android.media.RingtoneManager
import android.os.Build
import android.os.Bundle
import io.flutter.embedding.android.FlutterActivity

class MainActivity : FlutterActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        createEscalationChannel()
        createDigestChannel()
    }

    /// 서버(notify.AndroidChannel)가 위급 알림을 보내는 채널. 소리와 진동이 있는
    /// 높은 중요도로 만들어 방해 금지가 아닐 때 화면 위에 바로 뜨게 합니다.
    private fun createEscalationChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val channel = NotificationChannel(
            "escalation",
            "위급 알림",
            NotificationManager.IMPORTANCE_HIGH,
        ).apply {
            description = "어르신이 넘어짐·통증 등을 말씀하시면 알려드려요"
            enableVibration(true)
            setSound(
                RingtoneManager.getDefaultUri(RingtoneManager.TYPE_ALARM),
                AudioAttributes.Builder()
                    .setUsage(AudioAttributes.USAGE_NOTIFICATION_EVENT)
                    .build(),
            )
        }
        getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
    }

    /// 서버(notify.DigestChannel)가 하루 소식을 보내는 채널. 알람처럼 울리지
    /// 않도록 보통 중요도로 만듭니다.
    private fun createDigestChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val channel = NotificationChannel(
            "digest",
            "하루 소식",
            NotificationManager.IMPORTANCE_DEFAULT,
        ).apply {
            description = "어르신이 오늘 어떻게 지내셨는지 저녁에 알려드려요"
        }
        getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
    }
}
