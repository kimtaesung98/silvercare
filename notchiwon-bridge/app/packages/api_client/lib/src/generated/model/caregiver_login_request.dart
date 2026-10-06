//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class CaregiverLoginRequest {
  /// Returns a new [CaregiverLoginRequest] instance.
  CaregiverLoginRequest({
    required this.loginId,
    required this.password,
  });

  String loginId;

  String password;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CaregiverLoginRequest &&
          other.loginId == loginId &&
          other.password == password;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (loginId.hashCode) + (password.hashCode);

  @override
  String toString() =>
      'CaregiverLoginRequest[loginId=$loginId, password=$password]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'loginId'] = this.loginId;
    json[r'password'] = this.password;
    return json;
  }

  /// Returns a new [CaregiverLoginRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static CaregiverLoginRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'loginId'),
            'Required key "CaregiverLoginRequest[loginId]" is missing from JSON.');
        assert(json[r'loginId'] != null,
            'Required key "CaregiverLoginRequest[loginId]" has a null value in JSON.');
        assert(json.containsKey(r'password'),
            'Required key "CaregiverLoginRequest[password]" is missing from JSON.');
        assert(json[r'password'] != null,
            'Required key "CaregiverLoginRequest[password]" has a null value in JSON.');
        return true;
      }());

      return CaregiverLoginRequest(
        loginId: mapValueOfType<String>(json, r'loginId')!,
        password: mapValueOfType<String>(json, r'password')!,
      );
    }
    return null;
  }

  static List<CaregiverLoginRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CaregiverLoginRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CaregiverLoginRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, CaregiverLoginRequest> mapFromJson(dynamic json) {
    final map = <String, CaregiverLoginRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = CaregiverLoginRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of CaregiverLoginRequest-objects as value to a dart map
  static Map<String, List<CaregiverLoginRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<CaregiverLoginRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = CaregiverLoginRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'loginId',
    'password',
  };
}
