//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class GuardianLoginResponse {
  /// Returns a new [GuardianLoginResponse] instance.
  GuardianLoginResponse({
    required this.accessToken,
    required this.expiresAt,
    required this.guardian,
    this.elders = const [],
  });

  String accessToken;

  DateTime expiresAt;

  Guardian guardian;

  List<ElderSummary> elders;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is GuardianLoginResponse &&
          other.accessToken == accessToken &&
          other.expiresAt == expiresAt &&
          other.guardian == guardian &&
          _deepEquality.equals(other.elders, elders);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (accessToken.hashCode) +
      (expiresAt.hashCode) +
      (guardian.hashCode) +
      (elders.hashCode);

  @override
  String toString() =>
      'GuardianLoginResponse[accessToken=$accessToken, expiresAt=$expiresAt, guardian=$guardian, elders=$elders]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'accessToken'] = this.accessToken;
    json[r'expiresAt'] = this.expiresAt.toUtc().toIso8601String();
    json[r'guardian'] = this.guardian;
    json[r'elders'] = this.elders;
    return json;
  }

  /// Returns a new [GuardianLoginResponse] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static GuardianLoginResponse? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'accessToken'),
            'Required key "GuardianLoginResponse[accessToken]" is missing from JSON.');
        assert(json[r'accessToken'] != null,
            'Required key "GuardianLoginResponse[accessToken]" has a null value in JSON.');
        assert(json.containsKey(r'expiresAt'),
            'Required key "GuardianLoginResponse[expiresAt]" is missing from JSON.');
        assert(json[r'expiresAt'] != null,
            'Required key "GuardianLoginResponse[expiresAt]" has a null value in JSON.');
        assert(json.containsKey(r'guardian'),
            'Required key "GuardianLoginResponse[guardian]" is missing from JSON.');
        assert(json[r'guardian'] != null,
            'Required key "GuardianLoginResponse[guardian]" has a null value in JSON.');
        assert(json.containsKey(r'elders'),
            'Required key "GuardianLoginResponse[elders]" is missing from JSON.');
        assert(json[r'elders'] != null,
            'Required key "GuardianLoginResponse[elders]" has a null value in JSON.');
        return true;
      }());

      return GuardianLoginResponse(
        accessToken: mapValueOfType<String>(json, r'accessToken')!,
        expiresAt: mapDateTime(json, r'expiresAt', r'')!,
        guardian: Guardian.fromJson(json[r'guardian'])!,
        elders: ElderSummary.listFromJson(json[r'elders']),
      );
    }
    return null;
  }

  static List<GuardianLoginResponse> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <GuardianLoginResponse>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = GuardianLoginResponse.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, GuardianLoginResponse> mapFromJson(dynamic json) {
    final map = <String, GuardianLoginResponse>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = GuardianLoginResponse.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of GuardianLoginResponse-objects as value to a dart map
  static Map<String, List<GuardianLoginResponse>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<GuardianLoginResponse>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = GuardianLoginResponse.listFromJson(
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
    'guardian',
    'elders',
  };
}
