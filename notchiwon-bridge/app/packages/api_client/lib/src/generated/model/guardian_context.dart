//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class GuardianContext {
  /// Returns a new [GuardianContext] instance.
  GuardianContext({
    required this.guardian,
    this.elders = const [],
  });

  Guardian guardian;

  List<ElderSummary> elders;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is GuardianContext &&
          other.guardian == guardian &&
          _deepEquality.equals(other.elders, elders);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (guardian.hashCode) + (elders.hashCode);

  @override
  String toString() => 'GuardianContext[guardian=$guardian, elders=$elders]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'guardian'] = this.guardian;
    json[r'elders'] = this.elders;
    return json;
  }

  /// Returns a new [GuardianContext] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static GuardianContext? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'guardian'),
            'Required key "GuardianContext[guardian]" is missing from JSON.');
        assert(json[r'guardian'] != null,
            'Required key "GuardianContext[guardian]" has a null value in JSON.');
        assert(json.containsKey(r'elders'),
            'Required key "GuardianContext[elders]" is missing from JSON.');
        assert(json[r'elders'] != null,
            'Required key "GuardianContext[elders]" has a null value in JSON.');
        return true;
      }());

      return GuardianContext(
        guardian: Guardian.fromJson(json[r'guardian'])!,
        elders: ElderSummary.listFromJson(json[r'elders']),
      );
    }
    return null;
  }

  static List<GuardianContext> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <GuardianContext>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = GuardianContext.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, GuardianContext> mapFromJson(dynamic json) {
    final map = <String, GuardianContext>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = GuardianContext.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of GuardianContext-objects as value to a dart map
  static Map<String, List<GuardianContext>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<GuardianContext>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = GuardianContext.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'guardian',
    'elders',
  };
}
