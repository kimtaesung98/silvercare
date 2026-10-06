//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class VisitList {
  /// Returns a new [VisitList] instance.
  VisitList({
    this.visits = const [],
  });

  List<Visit> visits;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is VisitList && _deepEquality.equals(other.visits, visits);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (visits.hashCode);

  @override
  String toString() => 'VisitList[visits=$visits]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'visits'] = this.visits;
    return json;
  }

  /// Returns a new [VisitList] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static VisitList? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'visits'),
            'Required key "VisitList[visits]" is missing from JSON.');
        assert(json[r'visits'] != null,
            'Required key "VisitList[visits]" has a null value in JSON.');
        return true;
      }());

      return VisitList(
        visits: Visit.listFromJson(json[r'visits']),
      );
    }
    return null;
  }

  static List<VisitList> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <VisitList>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = VisitList.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, VisitList> mapFromJson(dynamic json) {
    final map = <String, VisitList>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = VisitList.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of VisitList-objects as value to a dart map
  static Map<String, List<VisitList>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<VisitList>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = VisitList.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'visits',
  };
}
