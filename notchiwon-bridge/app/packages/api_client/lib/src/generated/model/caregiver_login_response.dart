//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class CaregiverLoginResponse {
  /// Returns a new [CaregiverLoginResponse] instance.
  CaregiverLoginResponse({
    required this.accessToken,
    required this.expiresAt,
    required this.caregiver,
  });

  String accessToken;

  DateTime expiresAt;

  Caregiver caregiver;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CaregiverLoginResponse &&
          other.accessToken == accessToken &&
          other.expiresAt == expiresAt &&
          other.caregiver == caregiver;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (accessToken.hashCode) + (expiresAt.hashCode) + (caregiver.hashCode);

  @override
  String toString() =>
      'CaregiverLoginResponse[accessToken=$accessToken, expiresAt=$expiresAt, caregiver=$caregiver]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'accessToken'] = this.accessToken;
    json[r'expiresAt'] = this.expiresAt.toUtc().toIso8601String();
    json[r'caregiver'] = this.caregiver;
    return json;
  }

  /// Returns a new [CaregiverLoginResponse] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static CaregiverLoginResponse? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'accessToken'),
            'Required key "CaregiverLoginResponse[accessToken]" is missing from JSON.');
        assert(json[r'accessToken'] != null,
            'Required key "CaregiverLoginResponse[accessToken]" has a null value in JSON.');
        assert(json.containsKey(r'expiresAt'),
            'Required key "CaregiverLoginResponse[expiresAt]" is missing from JSON.');
        assert(json[r'expiresAt'] != null,
            'Required key "CaregiverLoginResponse[expiresAt]" has a null value in JSON.');
        assert(json.containsKey(r'caregiver'),
            'Required key "CaregiverLoginResponse[caregiver]" is missing from JSON.');
        assert(json[r'caregiver'] != null,
            'Required key "CaregiverLoginResponse[caregiver]" has a null value in JSON.');
        return true;
      }());

      return CaregiverLoginResponse(
        accessToken: mapValueOfType<String>(json, r'accessToken')!,
        expiresAt: mapDateTime(json, r'expiresAt', r'')!,
        caregiver: Caregiver.fromJson(json[r'caregiver'])!,
      );
    }
    return null;
  }

  static List<CaregiverLoginResponse> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CaregiverLoginResponse>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CaregiverLoginResponse.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, CaregiverLoginResponse> mapFromJson(dynamic json) {
    final map = <String, CaregiverLoginResponse>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = CaregiverLoginResponse.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of CaregiverLoginResponse-objects as value to a dart map
  static Map<String, List<CaregiverLoginResponse>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<CaregiverLoginResponse>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = CaregiverLoginResponse.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'accessToken',
    'expiresAt',
    'caregiver',
  };
}
