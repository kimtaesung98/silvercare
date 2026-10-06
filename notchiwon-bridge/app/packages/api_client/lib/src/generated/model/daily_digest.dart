//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class DailyDigest {
  /// Returns a new [DailyDigest] instance.
  DailyDigest({
    required this.elderId,
    required this.date,
    required this.summaryText,
    this.emotionFlag,
    required this.sessionCount,
    required this.escalationCount,
    required this.generatedAt,
    this.sentAt,
  });

  String elderId;

  DateTime date;

  String summaryText;

  String? emotionFlag;

  /// Minimum value: 0
  int sessionCount;

  /// Minimum value: 0
  int escalationCount;

  DateTime generatedAt;

  DateTime? sentAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DailyDigest &&
          other.elderId == elderId &&
          other.date == date &&
          other.summaryText == summaryText &&
          other.emotionFlag == emotionFlag &&
          other.sessionCount == sessionCount &&
          other.escalationCount == escalationCount &&
          other.generatedAt == generatedAt &&
          other.sentAt == sentAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (elderId.hashCode) +
      (date.hashCode) +
      (summaryText.hashCode) +
      (emotionFlag == null ? 0 : emotionFlag!.hashCode) +
      (sessionCount.hashCode) +
      (escalationCount.hashCode) +
      (generatedAt.hashCode) +
      (sentAt == null ? 0 : sentAt!.hashCode);

  @override
  String toString() =>
      'DailyDigest[elderId=$elderId, date=$date, summaryText=$summaryText, emotionFlag=$emotionFlag, sessionCount=$sessionCount, escalationCount=$escalationCount, generatedAt=$generatedAt, sentAt=$sentAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'elderId'] = this.elderId;
    json[r'date'] = _dateFormatter.format(this.date);
    json[r'summaryText'] = this.summaryText;
    if (this.emotionFlag != null) {
      json[r'emotionFlag'] = this.emotionFlag;
    } else {
      json[r'emotionFlag'] = null;
    }
    json[r'sessionCount'] = this.sessionCount;
    json[r'escalationCount'] = this.escalationCount;
    json[r'generatedAt'] = this.generatedAt.toUtc().toIso8601String();
    if (this.sentAt != null) {
      json[r'sentAt'] = this.sentAt!.toUtc().toIso8601String();
    } else {
      json[r'sentAt'] = null;
    }
    return json;
  }

  /// Returns a new [DailyDigest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DailyDigest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'elderId'),
            'Required key "DailyDigest[elderId]" is missing from JSON.');
        assert(json[r'elderId'] != null,
            'Required key "DailyDigest[elderId]" has a null value in JSON.');
        assert(json.containsKey(r'date'),
            'Required key "DailyDigest[date]" is missing from JSON.');
        assert(json[r'date'] != null,
            'Required key "DailyDigest[date]" has a null value in JSON.');
        assert(json.containsKey(r'summaryText'),
            'Required key "DailyDigest[summaryText]" is missing from JSON.');
        assert(json[r'summaryText'] != null,
            'Required key "DailyDigest[summaryText]" has a null value in JSON.');
        assert(json.containsKey(r'sessionCount'),
            'Required key "DailyDigest[sessionCount]" is missing from JSON.');
        assert(json[r'sessionCount'] != null,
            'Required key "DailyDigest[sessionCount]" has a null value in JSON.');
        assert(json.containsKey(r'escalationCount'),
            'Required key "DailyDigest[escalationCount]" is missing from JSON.');
        assert(json[r'escalationCount'] != null,
            'Required key "DailyDigest[escalationCount]" has a null value in JSON.');
        assert(json.containsKey(r'generatedAt'),
            'Required key "DailyDigest[generatedAt]" is missing from JSON.');
        assert(json[r'generatedAt'] != null,
            'Required key "DailyDigest[generatedAt]" has a null value in JSON.');
        return true;
      }());

      return DailyDigest(
        elderId: mapValueOfType<String>(json, r'elderId')!,
        date: mapDateTime(json, r'date', r'')!,
        summaryText: mapValueOfType<String>(json, r'summaryText')!,
        emotionFlag: mapValueOfType<String>(json, r'emotionFlag'),
        sessionCount: mapValueOfType<int>(json, r'sessionCount')!,
        escalationCount: mapValueOfType<int>(json, r'escalationCount')!,
        generatedAt: mapDateTime(json, r'generatedAt', r'')!,
        sentAt: mapDateTime(json, r'sentAt', r''),
      );
    }
    return null;
  }

  static List<DailyDigest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DailyDigest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DailyDigest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DailyDigest> mapFromJson(dynamic json) {
    final map = <String, DailyDigest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DailyDigest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DailyDigest-objects as value to a dart map
  static Map<String, List<DailyDigest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DailyDigest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DailyDigest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'elderId',
    'date',
    'summaryText',
    'sessionCount',
    'escalationCount',
    'generatedAt',
  };
}
