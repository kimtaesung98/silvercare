//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class Caregiver {
  /// Returns a new [Caregiver] instance.
  Caregiver({
    required this.id,
    required this.name,
    required this.centerId,
  });

  String id;

  String name;

  String centerId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Caregiver &&
          other.id == id &&
          other.name == name &&
          other.centerId == centerId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) + (name.hashCode) + (centerId.hashCode);

  @override
  String toString() => 'Caregiver[id=$id, name=$name, centerId=$centerId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'name'] = this.name;
    json[r'centerId'] = this.centerId;
    return json;
  }

  /// Returns a new [Caregiver] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Caregiver? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Caregiver[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Caregiver[id]" has a null value in JSON.');
        assert(json.containsKey(r'name'),
            'Required key "Caregiver[name]" is missing from JSON.');
        assert(json[r'name'] != null,
            'Required key "Caregiver[name]" has a null value in JSON.');
        assert(json.containsKey(r'centerId'),
            'Required key "Caregiver[centerId]" is missing from JSON.');
        assert(json[r'centerId'] != null,
            'Required key "Caregiver[centerId]" has a null value in JSON.');
        return true;
      }());

      return Caregiver(
        id: mapValueOfType<String>(json, r'id')!,
        name: mapValueOfType<String>(json, r'name')!,
        centerId: mapValueOfType<String>(json, r'centerId')!,
      );
    }
    return null;
  }

  static List<Caregiver> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Caregiver>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Caregiver.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Caregiver> mapFromJson(dynamic json) {
    final map = <String, Caregiver>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Caregiver.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Caregiver-objects as value to a dart map
  static Map<String, List<Caregiver>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Caregiver>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Caregiver.listFromJson(
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
    'name',
    'centerId',
  };
}
