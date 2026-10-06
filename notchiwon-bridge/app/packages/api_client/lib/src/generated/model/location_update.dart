//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class LocationUpdate {
  /// Returns a new [LocationUpdate] instance.
  LocationUpdate({
    required this.latitude,
    required this.longitude,
    this.accuracyM,
    required this.recordedAt,
  });

  /// Minimum value: -90
  /// Maximum value: 90
  double latitude;

  /// Minimum value: -180
  /// Maximum value: 180
  double longitude;

  /// 위치 정확도(미터)
  ///
  /// Minimum value: 0
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  double? accuracyM;

  /// 휴대폰에서 위치를 잰 시각
  DateTime recordedAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is LocationUpdate &&
          other.latitude == latitude &&
          other.longitude == longitude &&
          other.accuracyM == accuracyM &&
          other.recordedAt == recordedAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (latitude.hashCode) +
      (longitude.hashCode) +
      (accuracyM == null ? 0 : accuracyM!.hashCode) +
      (recordedAt.hashCode);

  @override
  String toString() =>
      'LocationUpdate[latitude=$latitude, longitude=$longitude, accuracyM=$accuracyM, recordedAt=$recordedAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'latitude'] = this.latitude;
    json[r'longitude'] = this.longitude;
    if (this.accuracyM != null) {
      json[r'accuracyM'] = this.accuracyM;
    } else {
      json[r'accuracyM'] = null;
    }
    json[r'recordedAt'] = this.recordedAt.toUtc().toIso8601String();
    return json;
  }

  /// Returns a new [LocationUpdate] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static LocationUpdate? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'latitude'),
            'Required key "LocationUpdate[latitude]" is missing from JSON.');
        assert(json[r'latitude'] != null,
            'Required key "LocationUpdate[latitude]" has a null value in JSON.');
        assert(json.containsKey(r'longitude'),
            'Required key "LocationUpdate[longitude]" is missing from JSON.');
        assert(json[r'longitude'] != null,
            'Required key "LocationUpdate[longitude]" has a null value in JSON.');
        assert(json.containsKey(r'recordedAt'),
            'Required key "LocationUpdate[recordedAt]" is missing from JSON.');
        assert(json[r'recordedAt'] != null,
            'Required key "LocationUpdate[recordedAt]" has a null value in JSON.');
        return true;
      }());

      return LocationUpdate(
        latitude: mapValueOfType<double>(json, r'latitude')!,
        longitude: mapValueOfType<double>(json, r'longitude')!,
        accuracyM: mapValueOfType<double>(json, r'accuracyM'),
        recordedAt: mapDateTime(json, r'recordedAt', r'')!,
      );
    }
    return null;
  }

  static List<LocationUpdate> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <LocationUpdate>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = LocationUpdate.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, LocationUpdate> mapFromJson(dynamic json) {
    final map = <String, LocationUpdate>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = LocationUpdate.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of LocationUpdate-objects as value to a dart map
  static Map<String, List<LocationUpdate>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<LocationUpdate>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = LocationUpdate.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'latitude',
    'longitude',
    'recordedAt',
  };
}
