//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class Briefing {
  /// Returns a new [Briefing] instance.
  Briefing({
    required this.sessionId,
    required this.summaryText,
    this.topKeywords = const [],
    this.emotionFlag,
    this.overallEmotionTag,
    this.escalationCount,
    required this.generatedAt,
    this.readByCaregiverAt,
  });

  String sessionId;

  String summaryText;

  List<String> topKeywords;

  /// 평소와 다른 점 한 줄 (없으면 null)
  String? emotionFlag;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  EmotionTag? overallEmotionTag;

  /// Minimum value: 0
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? escalationCount;

  DateTime generatedAt;

  DateTime? readByCaregiverAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Briefing &&
          other.sessionId == sessionId &&
          other.summaryText == summaryText &&
          _deepEquality.equals(other.topKeywords, topKeywords) &&
          other.emotionFlag == emotionFlag &&
          other.overallEmotionTag == overallEmotionTag &&
          other.escalationCount == escalationCount &&
          other.generatedAt == generatedAt &&
          other.readByCaregiverAt == readByCaregiverAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (sessionId.hashCode) +
      (summaryText.hashCode) +
      (topKeywords.hashCode) +
      (emotionFlag == null ? 0 : emotionFlag!.hashCode) +
      (overallEmotionTag == null ? 0 : overallEmotionTag!.hashCode) +
      (escalationCount == null ? 0 : escalationCount!.hashCode) +
      (generatedAt.hashCode) +
      (readByCaregiverAt == null ? 0 : readByCaregiverAt!.hashCode);

  @override
  String toString() =>
      'Briefing[sessionId=$sessionId, summaryText=$summaryText, topKeywords=$topKeywords, emotionFlag=$emotionFlag, overallEmotionTag=$overallEmotionTag, escalationCount=$escalationCount, generatedAt=$generatedAt, readByCaregiverAt=$readByCaregiverAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'sessionId'] = this.sessionId;
    json[r'summaryText'] = this.summaryText;
    json[r'topKeywords'] = this.topKeywords;
    if (this.emotionFlag != null) {
      json[r'emotionFlag'] = this.emotionFlag;
    } else {
      json[r'emotionFlag'] = null;
    }
    if (this.overallEmotionTag != null) {
      json[r'overallEmotionTag'] = this.overallEmotionTag;
    } else {
      json[r'overallEmotionTag'] = null;
    }
    if (this.escalationCount != null) {
      json[r'escalationCount'] = this.escalationCount;
    } else {
      json[r'escalationCount'] = null;
    }
    json[r'generatedAt'] = this.generatedAt.toUtc().toIso8601String();
    if (this.readByCaregiverAt != null) {
      json[r'readByCaregiverAt'] =
          this.readByCaregiverAt!.toUtc().toIso8601String();
    } else {
      json[r'readByCaregiverAt'] = null;
    }
    return json;
  }

  /// Returns a new [Briefing] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Briefing? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'sessionId'),
            'Required key "Briefing[sessionId]" is missing from JSON.');
        assert(json[r'sessionId'] != null,
            'Required key "Briefing[sessionId]" has a null value in JSON.');
        assert(json.containsKey(r'summaryText'),
            'Required key "Briefing[summaryText]" is missing from JSON.');
        assert(json[r'summaryText'] != null,
            'Required key "Briefing[summaryText]" has a null value in JSON.');
        assert(json.containsKey(r'topKeywords'),
            'Required key "Briefing[topKeywords]" is missing from JSON.');
        assert(json[r'topKeywords'] != null,
            'Required key "Briefing[topKeywords]" has a null value in JSON.');
        assert(json.containsKey(r'generatedAt'),
            'Required key "Briefing[generatedAt]" is missing from JSON.');
        assert(json[r'generatedAt'] != null,
            'Required key "Briefing[generatedAt]" has a null value in JSON.');
        return true;
      }());

      return Briefing(
        sessionId: mapValueOfType<String>(json, r'sessionId')!,
        summaryText: mapValueOfType<String>(json, r'summaryText')!,
        topKeywords: json[r'topKeywords'] is Iterable
            ? (json[r'topKeywords'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
        emotionFlag: mapValueOfType<String>(json, r'emotionFlag'),
        overallEmotionTag: EmotionTag.fromJson(json[r'overallEmotionTag']),
        escalationCount: mapValueOfType<int>(json, r'escalationCount'),
        generatedAt: mapDateTime(json, r'generatedAt', r'')!,
        readByCaregiverAt: mapDateTime(json, r'readByCaregiverAt', r''),
      );
    }
    return null;
  }

  static List<Briefing> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Briefing>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Briefing.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Briefing> mapFromJson(dynamic json) {
    final map = <String, Briefing>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Briefing.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Briefing-objects as value to a dart map
  static Map<String, List<Briefing>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Briefing>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Briefing.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'sessionId',
    'summaryText',
    'topKeywords',
    'generatedAt',
  };
}
