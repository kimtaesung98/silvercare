//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class Place {
  /// Returns a new [Place] instance.
  Place({
    required this.latitude,
    required this.longitude,
    this.address,
  });

  double latitude;

  double longitude;

  String? address;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Place &&
          other.latitude == latitude &&
          other.longitude == longitude &&
          other.address == address;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (latitude.hashCode) +
      (longitude.hashCode) +
      (address == null ? 0 : address!.hashCode);

  @override
  String toString() =>
      'Place[latitude=$latitude, longitude=$longitude, address=$address]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'latitude'] = this.latitude;
    json[r'longitude'] = this.longitude;
    if (this.address != null) {
      json[r'address'] = this.address;
    } else {
      json[r'address'] = null;
    }
    return json;
  }

  /// Returns a new [Place] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Place? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'latitude'),
            'Required key "Place[latitude]" is missing from JSON.');
        assert(json[r'latitude'] != null,
            'Required key "Place[latitude]" has a null value in JSON.');
        assert(json.containsKey(r'longitude'),
            'Required key "Place[longitude]" is missing from JSON.');
        assert(json[r'longitude'] != null,
            'Required key "Place[longitude]" has a null value in JSON.');
        return true;
      }());

      return Place(
        latitude: mapValueOfType<double>(json, r'latitude')!,
        longitude: mapValueOfType<double>(json, r'longitude')!,
        address: mapValueOfType<String>(json, r'address'),
      );
    }
    return null;
  }

  static List<Place> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Place>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Place.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Place> mapFromJson(dynamic json) {
    final map = <String, Place>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Place.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Place-objects as value to a dart map
  static Map<String, List<Place>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Place>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Place.listFromJson(
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
  };
}
