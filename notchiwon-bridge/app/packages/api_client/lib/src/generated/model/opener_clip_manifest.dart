//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class OpenerClipManifest {
  /// Returns a new [OpenerClipManifest] instance.
  OpenerClipManifest({
    required this.voice,
    required this.version,
    this.clips = const [],
  });

  String voice;

  /// 목록 내용의 해시. 바뀌면 캐시를 갱신합니다.
  String version;

  List<OpenerClip> clips;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is OpenerClipManifest &&
          other.voice == voice &&
          other.version == version &&
          _deepEquality.equals(other.clips, clips);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (voice.hashCode) + (version.hashCode) + (clips.hashCode);

  @override
  String toString() =>
      'OpenerClipManifest[voice=$voice, version=$version, clips=$clips]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'voice'] = this.voice;
    json[r'version'] = this.version;
    json[r'clips'] = this.clips;
    return json;
  }

  /// Returns a new [OpenerClipManifest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static OpenerClipManifest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'voice'),
            'Required key "OpenerClipManifest[voice]" is missing from JSON.');
        assert(json[r'voice'] != null,
            'Required key "OpenerClipManifest[voice]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "OpenerClipManifest[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "OpenerClipManifest[version]" has a null value in JSON.');
        assert(json.containsKey(r'clips'),
            'Required key "OpenerClipManifest[clips]" is missing from JSON.');
        assert(json[r'clips'] != null,
            'Required key "OpenerClipManifest[clips]" has a null value in JSON.');
        return true;
      }());

      return OpenerClipManifest(
        voice: mapValueOfType<String>(json, r'voice')!,
        version: mapValueOfType<String>(json, r'version')!,
        clips: OpenerClip.listFromJson(json[r'clips']),
      );
    }
    return null;
  }

  static List<OpenerClipManifest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <OpenerClipManifest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = OpenerClipManifest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, OpenerClipManifest> mapFromJson(dynamic json) {
    final map = <String, OpenerClipManifest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = OpenerClipManifest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of OpenerClipManifest-objects as value to a dart map
  static Map<String, List<OpenerClipManifest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<OpenerClipManifest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = OpenerClipManifest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'voice',
    'version',
    'clips',
  };
}
