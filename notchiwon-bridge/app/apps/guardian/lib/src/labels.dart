import 'package:api_client/api_client.dart';

/// 서버의 `notify.TriggerLabels`와 같은 말을 씁니다.
String triggerLabel(EscalationTriggerType t) => switch (t) {
  EscalationTriggerType.FALL_MENTION => '낙상 언급',
  EscalationTriggerType.PAIN_COMPLAINT => '통증 호소',
  EscalationTriggerType.SELF_OR_OTHER_HARM => '자해·타해 언급',
  EscalationTriggerType.OTHER_ANOMALY => '이상 징후',
};

/// `오후 3:05`처럼 기기 시간대의 시각으로 씁니다.
String clock(DateTime t) {
  final l = t.toLocal();
  final h = l.hour % 12 == 0 ? 12 : l.hour % 12;
  final m = l.minute.toString().padLeft(2, '0');
  return '${l.hour < 12 ? '오전' : '오후'} $h:$m';
}

/// `10월 6일 (월)`처럼 하루 소식의 날짜로 씁니다.
String day(DateTime d) {
  const names = ['월', '화', '수', '목', '금', '토', '일'];
  return '${d.month}월 ${d.day}일 (${names[d.weekday - 1]})';
}

/// `어제`, `오늘`처럼 가까운 날은 말로 씁니다.
String dayOrToday(DateTime d, DateTime now) {
  final days = DateTime(
    d.year,
    d.month,
    d.day,
  ).difference(DateTime(now.year, now.month, now.day)).inDays;
  return switch (days) {
    0 => '오늘',
    -1 => '어제',
    _ => day(d),
  };
}

/// 안부 대화 시각 목록을 읽을 말로. 비어 있으면 어르신이 먼저 말을 거실 때만 대화합니다.
String checkInLabel(List<String> times) =>
    times.isEmpty ? '정해진 시각 없음' : times.join(', ');
