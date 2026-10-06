//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class ActiveSession {
  /// Returns a new [ActiveSession] instance.
  ActiveSession({
    required this.id,
    required this.mode,
    required this.startedAt,
  });

  String id;

  SessionMode mode;

  DateTime startedAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ActiveSession &&
          other.id == id &&
          other.mode == mode &&
          other.startedAt == startedAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) + (mode.hashCode) + (startedAt.hashCode);

  @override
  String toString() =>
      'ActiveSession[id=$id, mode=$mode, startedAt=$startedAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'mode'] = this.mode;
    json[r'startedAt'] = this.startedAt.toUtc().toIso8601String();
    return json;
  }

  /// Returns a new [ActiveSession] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ActiveSession? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "ActiveSession[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "ActiveSession[id]" has a null value in JSON.');
        assert(json.containsKey(r'mode'),
            'Required key "ActiveSession[mode]" is missing from JSON.');
        assert(json[r'mode'] != null,
            'Required key "ActiveSession[mode]" has a null value in JSON.');
        assert(json.containsKey(r'startedAt'),
            'Required key "ActiveSession[startedAt]" is missing from JSON.');
        assert(json[r'startedAt'] != null,
            'Required key "ActiveSession[startedAt]" has a null value in JSON.');
        return true;
      }());

      return ActiveSession(
        id: mapValueOfType<String>(json, r'id')!,
        mode: SessionMode.fromJson(json[r'mode'])!,
        startedAt: mapDateTime(json, r'startedAt', r'')!,
      );
    }
    return null;
  }

  static List<ActiveSession> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ActiveSession>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ActiveSession.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ActiveSession> mapFromJson(dynamic json) {
    final map = <String, ActiveSession>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ActiveSession.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ActiveSession-objects as value to a dart map
  static Map<String, List<ActiveSession>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ActiveSession>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ActiveSession.listFromJson(
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
    'mode',
    'startedAt',
  };
}
