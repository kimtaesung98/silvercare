//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class Visit {
  /// Returns a new [Visit] instance.
  Visit({
    required this.id,
    required this.elder,
    required this.scheduledTime,
    this.etaCurrent,
    required this.status,
    this.actualArrivalTime,
    this.sessionId,
    this.destination,
  });

  String id;

  ElderSummary elder;

  DateTime scheduledTime;

  DateTime? etaCurrent;

  VisitStatus status;

  DateTime? actualArrivalTime;

  /// 픽업 대기 세션이 시작됐으면 그 ID
  String? sessionId;

  /// 어르신 댁 (등록되지 않았으면 null)
  Place? destination;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Visit &&
          other.id == id &&
          other.elder == elder &&
          other.scheduledTime == scheduledTime &&
          other.etaCurrent == etaCurrent &&
          other.status == status &&
          other.actualArrivalTime == actualArrivalTime &&
          other.sessionId == sessionId &&
          other.destination == destination;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (elder.hashCode) +
      (scheduledTime.hashCode) +
      (etaCurrent == null ? 0 : etaCurrent!.hashCode) +
      (status.hashCode) +
      (actualArrivalTime == null ? 0 : actualArrivalTime!.hashCode) +
      (sessionId == null ? 0 : sessionId!.hashCode) +
      (destination == null ? 0 : destination!.hashCode);

  @override
  String toString() =>
      'Visit[id=$id, elder=$elder, scheduledTime=$scheduledTime, etaCurrent=$etaCurrent, status=$status, actualArrivalTime=$actualArrivalTime, sessionId=$sessionId, destination=$destination]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'elder'] = this.elder;
    json[r'scheduledTime'] = this.scheduledTime.toUtc().toIso8601String();
    if (this.etaCurrent != null) {
      json[r'etaCurrent'] = this.etaCurrent!.toUtc().toIso8601String();
    } else {
      json[r'etaCurrent'] = null;
    }
    json[r'status'] = this.status;
    if (this.actualArrivalTime != null) {
      json[r'actualArrivalTime'] =
          this.actualArrivalTime!.toUtc().toIso8601String();
    } else {
      json[r'actualArrivalTime'] = null;
    }
    if (this.sessionId != null) {
      json[r'sessionId'] = this.sessionId;
    } else {
      json[r'sessionId'] = null;
    }
    if (this.destination != null) {
      json[r'destination'] = this.destination;
    } else {
      json[r'destination'] = null;
    }
    return json;
  }

  /// Returns a new [Visit] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Visit? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Visit[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Visit[id]" has a null value in JSON.');
        assert(json.containsKey(r'elder'),
            'Required key "Visit[elder]" is missing from JSON.');
        assert(json[r'elder'] != null,
            'Required key "Visit[elder]" has a null value in JSON.');
        assert(json.containsKey(r'scheduledTime'),
            'Required key "Visit[scheduledTime]" is missing from JSON.');
        assert(json[r'scheduledTime'] != null,
            'Required key "Visit[scheduledTime]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "Visit[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "Visit[status]" has a null value in JSON.');
        return true;
      }());

      return Visit(
        id: mapValueOfType<String>(json, r'id')!,
        elder: ElderSummary.fromJson(json[r'elder'])!,
        scheduledTime: mapDateTime(json, r'scheduledTime', r'')!,
        etaCurrent: mapDateTime(json, r'etaCurrent', r''),
        status: VisitStatus.fromJson(json[r'status'])!,
        actualArrivalTime: mapDateTime(json, r'actualArrivalTime', r''),
        sessionId: mapValueOfType<String>(json, r'sessionId'),
        destination: Place.fromJson(json[r'destination']),
      );
    }
    return null;
  }

  static List<Visit> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Visit>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Visit.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Visit> mapFromJson(dynamic json) {
    final map = <String, Visit>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Visit.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Visit-objects as value to a dart map
  static Map<String, List<Visit>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Visit>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Visit.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'id',
    'elder',
    'scheduledTime',
    'status',
  };
}
