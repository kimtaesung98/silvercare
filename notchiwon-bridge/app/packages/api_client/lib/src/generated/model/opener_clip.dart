//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class OpenerClip {
  /// Returns a new [OpenerClip] instance.
  OpenerClip({
    required this.id,
    required this.category,
    required this.text,
    required this.durationMs,
    required this.audioUrl,
  });

  String id;

  OpenerCategory category;

  String text;

  /// Minimum value: 1
  int durationMs;

  /// 음성 파일 경로 (`/tablet/opener-clips/{clipId}/audio`)
  String audioUrl;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is OpenerClip &&
          other.id == id &&
          other.category == category &&
          other.text == text &&
          other.durationMs == durationMs &&
          other.audioUrl == audioUrl;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (category.hashCode) +
      (text.hashCode) +
      (durationMs.hashCode) +
      (audioUrl.hashCode);

  @override
  String toString() =>
      'OpenerClip[id=$id, category=$category, text=$text, durationMs=$durationMs, audioUrl=$audioUrl]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'category'] = this.category;
    json[r'text'] = this.text;
    json[r'durationMs'] = this.durationMs;
    json[r'audioUrl'] = this.audioUrl;
    return json;
  }

  /// Returns a new [OpenerClip] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static OpenerClip? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "OpenerClip[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "OpenerClip[id]" has a null value in JSON.');
        assert(json.containsKey(r'category'),
            'Required key "OpenerClip[category]" is missing from JSON.');
        assert(json[r'category'] != null,
            'Required key "OpenerClip[category]" has a null value in JSON.');
        assert(json.containsKey(r'text'),
            'Required key "OpenerClip[text]" is missing from JSON.');
        assert(json[r'text'] != null,
            'Required key "OpenerClip[text]" has a null value in JSON.');
        assert(json.containsKey(r'durationMs'),
            'Required key "OpenerClip[durationMs]" is missing from JSON.');
        assert(json[r'durationMs'] != null,
            'Required key "OpenerClip[durationMs]" has a null value in JSON.');
        assert(json.containsKey(r'audioUrl'),
            'Required key "OpenerClip[audioUrl]" is missing from JSON.');
        assert(json[r'audioUrl'] != null,
            'Required key "OpenerClip[audioUrl]" has a null value in JSON.');
        return true;
      }());

      return OpenerClip(
        id: mapValueOfType<String>(json, r'id')!,
        category: OpenerCategory.fromJson(json[r'category'])!,
        text: mapValueOfType<String>(json, r'text')!,
        durationMs: mapValueOfType<int>(json, r'durationMs')!,
        audioUrl: mapValueOfType<String>(json, r'audioUrl')!,
      );
    }
    return null;
  }

  static List<OpenerClip> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <OpenerClip>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = OpenerClip.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, OpenerClip> mapFromJson(dynamic json) {
    final map = <String, OpenerClip>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = OpenerClip.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of OpenerClip-objects as value to a dart map
  static Map<String, List<OpenerClip>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<OpenerClip>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = OpenerClip.listFromJson(
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
    'category',
    'text',
    'durationMs',
    'audioUrl',
  };
}
