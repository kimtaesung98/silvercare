//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class CompanionSchedule {
  /// Returns a new [CompanionSchedule] instance.
  CompanionSchedule({
    required this.enabled,
    this.checkInTimes = const [],
    this.bedtimeStart,
    this.bedtimeEnd,
    required this.timeZone,
    this.dailyTokenLimit,
  });

  bool enabled;

  /// 보호자가 정한 안부 대화 시각 (현지 시각 `HH:MM`)
  List<String> checkInTimes;

  String? bedtimeStart;

  String? bedtimeEnd;

  String timeZone;

  /// 어르신별 하루 토큰 한도. null이면 서버 기본값
  ///
  /// Minimum value: 1
  int? dailyTokenLimit;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CompanionSchedule &&
          other.enabled == enabled &&
          _deepEquality.equals(other.checkInTimes, checkInTimes) &&
          other.bedtimeStart == bedtimeStart &&
          other.bedtimeEnd == bedtimeEnd &&
          other.timeZone == timeZone &&
          other.dailyTokenLimit == dailyTokenLimit;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (enabled.hashCode) +
      (checkInTimes.hashCode) +
      (bedtimeStart == null ? 0 : bedtimeStart!.hashCode) +
      (bedtimeEnd == null ? 0 : bedtimeEnd!.hashCode) +
      (timeZone.hashCode) +
      (dailyTokenLimit == null ? 0 : dailyTokenLimit!.hashCode);

  @override
  String toString() =>
      'CompanionSchedule[enabled=$enabled, checkInTimes=$checkInTimes, bedtimeStart=$bedtimeStart, bedtimeEnd=$bedtimeEnd, timeZone=$timeZone, dailyTokenLimit=$dailyTokenLimit]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'enabled'] = this.enabled;
    json[r'checkInTimes'] = this.checkInTimes;
    if (this.bedtimeStart != null) {
      json[r'bedtimeStart'] = this.bedtimeStart;
    } else {
      json[r'bedtimeStart'] = null;
    }
    if (this.bedtimeEnd != null) {
      json[r'bedtimeEnd'] = this.bedtimeEnd;
    } else {
      json[r'bedtimeEnd'] = null;
    }
    json[r'timeZone'] = this.timeZone;
    if (this.dailyTokenLimit != null) {
      json[r'dailyTokenLimit'] = this.dailyTokenLimit;
    } else {
      json[r'dailyTokenLimit'] = null;
    }
    return json;
  }

  /// Returns a new [CompanionSchedule] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static CompanionSchedule? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'enabled'),
            'Required key "CompanionSchedule[enabled]" is missing from JSON.');
        assert(json[r'enabled'] != null,
            'Required key "CompanionSchedule[enabled]" has a null value in JSON.');
        assert(json.containsKey(r'checkInTimes'),
            'Required key "CompanionSchedule[checkInTimes]" is missing from JSON.');
        assert(json[r'checkInTimes'] != null,
            'Required key "CompanionSchedule[checkInTimes]" has a null value in JSON.');
        assert(json.containsKey(r'timeZone'),
            'Required key "CompanionSchedule[timeZone]" is missing from JSON.');
        assert(json[r'timeZone'] != null,
            'Required key "CompanionSchedule[timeZone]" has a null value in JSON.');
        return true;
      }());

      return CompanionSchedule(
        enabled: mapValueOfType<bool>(json, r'enabled')!,
        checkInTimes: json[r'checkInTimes'] is Iterable
            ? (json[r'checkInTimes'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
        bedtimeStart: mapValueOfType<String>(json, r'bedtimeStart'),
        bedtimeEnd: mapValueOfType<String>(json, r'bedtimeEnd'),
        timeZone: mapValueOfType<String>(json, r'timeZone')!,
        dailyTokenLimit: mapValueOfType<int>(json, r'dailyTokenLimit'),
      );
    }
    return null;
  }

  static List<CompanionSchedule> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CompanionSchedule>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CompanionSchedule.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, CompanionSchedule> mapFromJson(dynamic json) {
    final map = <String, CompanionSchedule>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = CompanionSchedule.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of CompanionSchedule-objects as value to a dart map
  static Map<String, List<CompanionSchedule>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<CompanionSchedule>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = CompanionSchedule.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'enabled',
    'checkInTimes',
    'timeZone',
  };
}
