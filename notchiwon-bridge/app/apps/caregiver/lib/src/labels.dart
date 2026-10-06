import 'package:api_client/api_client.dart';

/// 서버의 `notify.TriggerLabels`와 같은 말을 씁니다.
String triggerLabel(EscalationTriggerType t) => switch (t) {
  EscalationTriggerType.FALL_MENTION => '낙상 언급',
  EscalationTriggerType.PAIN_COMPLAINT => '통증 호소',
  EscalationTriggerType.SELF_OR_OTHER_HARM => '자해·타해 언급',
  EscalationTriggerType.OTHER_ANOMALY => '이상 징후',
};

String visitStatusLabel(VisitStatus s) => switch (s) {
  VisitStatus.SCHEDULED => '예정',
  VisitStatus.EN_ROUTE => '이동 중',
  VisitStatus.SESSION_ACTIVE => '어르신과 대화 중',
  VisitStatus.COMPLETED => '도착 완료',
  VisitStatus.CANCELLED => '취소',
};

String emotionLabel(EmotionTag t) => switch (t) {
  EmotionTag.STABLE => '안정',
  EmotionTag.SLIGHTLY_ANXIOUS => '약간 불안',
  EmotionTag.UNUSUAL => '평소와 다름',
};

/// 위치를 보내거나 도착 처리할 수 있는 방문인지.
bool isOpen(VisitStatus s) =>
    s == VisitStatus.SCHEDULED ||
    s == VisitStatus.EN_ROUTE ||
    s == VisitStatus.SESSION_ACTIVE;

/// `오후 3:05`처럼 기기 시간대의 시각으로 씁니다.
String clock(DateTime t) {
  final l = t.toLocal();
  final h = l.hour % 12 == 0 ? 12 : l.hour % 12;
  final m = l.minute.toString().padLeft(2, '0');
  return '${l.hour < 12 ? '오전' : '오후'} $h:$m';
}

/// ETA 시각까지 남은 분. 이미 지났으면 0입니다.
int minutesUntil(DateTime eta, DateTime now) {
  final s = eta.difference(now).inSeconds;
  return s <= 0 ? 0 : (s + 59) ~/ 60;
}
