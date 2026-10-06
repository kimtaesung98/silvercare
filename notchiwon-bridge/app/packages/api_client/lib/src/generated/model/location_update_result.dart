//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class LocationUpdateResult {
  /// Returns a new [LocationUpdateResult] instance.
  LocationUpdateResult({
    required this.visitId,
    required this.status,
    required this.etaMinutes,
    required this.sessionStarted,
    this.sessionId,
  });

  String visitId;

  VisitStatus status;

  /// Minimum value: 0
  int etaMinutes;

  /// 이번 요청으로 세션이 시작됐으면 true (같은 방문에서 한 번만 true)
  bool sessionStarted;

  String? sessionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is LocationUpdateResult &&
          other.visitId == visitId &&
          other.status == status &&
          other.etaMinutes == etaMinutes &&
          other.sessionStarted == sessionStarted &&
          other.sessionId == sessionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (visitId.hashCode) +
      (status.hashCode) +
      (etaMinutes.hashCode) +
      (sessionStarted.hashCode) +
      (sessionId == null ? 0 : sessionId!.hashCode);

  @override
  String toString() =>
      'LocationUpdateResult[visitId=$visitId, status=$status, etaMinutes=$etaMinutes, sessionStarted=$sessionStarted, sessionId=$sessionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'visitId'] = this.visitId;
    json[r'status'] = this.status;
    json[r'etaMinutes'] = this.etaMinutes;
    json[r'sessionStarted'] = this.sessionStarted;
    if (this.sessionId != null) {
      json[r'sessionId'] = this.sessionId;
    } else {
      json[r'sessionId'] = null;
    }
    return json;
  }

  /// Returns a new [LocationUpdateResult] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static LocationUpdateResult? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'visitId'),
            'Required key "LocationUpdateResult[visitId]" is missing from JSON.');
        assert(json[r'visitId'] != null,
            'Required key "LocationUpdateResult[visitId]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "LocationUpdateResult[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "LocationUpdateResult[status]" has a null value in JSON.');
        assert(json.containsKey(r'etaMinutes'),
            'Required key "LocationUpdateResult[etaMinutes]" is missing from JSON.');
        assert(json[r'etaMinutes'] != null,
            'Required key "LocationUpdateResult[etaMinutes]" has a null value in JSON.');
        assert(json.containsKey(r'sessionStarted'),
            'Required key "LocationUpdateResult[sessionStarted]" is missing from JSON.');
        assert(json[r'sessionStarted'] != null,
            'Required key "LocationUpdateResult[sessionStarted]" has a null value in JSON.');
        return true;
      }());

      return LocationUpdateResult(
        visitId: mapValueOfType<String>(json, r'visitId')!,
        status: VisitStatus.fromJson(json[r'status'])!,
        etaMinutes: mapValueOfType<int>(json, r'etaMinutes')!,
        sessionStarted: mapValueOfType<bool>(json, r'sessionStarted')!,
        sessionId: mapValueOfType<String>(json, r'sessionId'),
      );
    }
    return null;
  }

  static List<LocationUpdateResult> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <LocationUpdateResult>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = LocationUpdateResult.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, LocationUpdateResult> mapFromJson(dynamic json) {
    final map = <String, LocationUpdateResult>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = LocationUpdateResult.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of LocationUpdateResult-objects as value to a dart map
  static Map<String, List<LocationUpdateResult>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<LocationUpdateResult>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = LocationUpdateResult.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'visitId',
    'status',
    'etaMinutes',
    'sessionStarted',
  };
}
