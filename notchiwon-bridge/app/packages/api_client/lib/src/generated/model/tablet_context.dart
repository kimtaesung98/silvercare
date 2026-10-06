//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class TabletContext {
  /// Returns a new [TabletContext] instance.
  TabletContext({
    required this.deviceId,
    required this.elder,
    this.preferredTtsVoice,
    required this.companionEnabled,
    this.activeSession,
  });

  String deviceId;

  ElderSummary elder;

  String? preferredTtsVoice;

  /// 말동무 버튼을 보여줄지
  bool companionEnabled;

  ActiveSession? activeSession;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TabletContext &&
          other.deviceId == deviceId &&
          other.elder == elder &&
          other.preferredTtsVoice == preferredTtsVoice &&
          other.companionEnabled == companionEnabled &&
          other.activeSession == activeSession;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (deviceId.hashCode) +
      (elder.hashCode) +
      (preferredTtsVoice == null ? 0 : preferredTtsVoice!.hashCode) +
      (companionEnabled.hashCode) +
      (activeSession == null ? 0 : activeSession!.hashCode);

  @override
  String toString() =>
      'TabletContext[deviceId=$deviceId, elder=$elder, preferredTtsVoice=$preferredTtsVoice, companionEnabled=$companionEnabled, activeSession=$activeSession]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'deviceId'] = this.deviceId;
    json[r'elder'] = this.elder;
    if (this.preferredTtsVoice != null) {
      json[r'preferredTtsVoice'] = this.preferredTtsVoice;
    } else {
      json[r'preferredTtsVoice'] = null;
    }
    json[r'companionEnabled'] = this.companionEnabled;
    if (this.activeSession != null) {
      json[r'activeSession'] = this.activeSession;
    } else {
      json[r'activeSession'] = null;
    }
    return json;
  }

  /// Returns a new [TabletContext] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TabletContext? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'deviceId'),
            'Required key "TabletContext[deviceId]" is missing from JSON.');
        assert(json[r'deviceId'] != null,
            'Required key "TabletContext[deviceId]" has a null value in JSON.');
        assert(json.containsKey(r'elder'),
            'Required key "TabletContext[elder]" is missing from JSON.');
        assert(json[r'elder'] != null,
            'Required key "TabletContext[elder]" has a null value in JSON.');
        assert(json.containsKey(r'companionEnabled'),
            'Required key "TabletContext[companionEnabled]" is missing from JSON.');
        assert(json[r'companionEnabled'] != null,
            'Required key "TabletContext[companionEnabled]" has a null value in JSON.');
        return true;
      }());

      return TabletContext(
        deviceId: mapValueOfType<String>(json, r'deviceId')!,
        elder: ElderSummary.fromJson(json[r'elder'])!,
        preferredTtsVoice: mapValueOfType<String>(json, r'preferredTtsVoice'),
        companionEnabled: mapValueOfType<bool>(json, r'companionEnabled')!,
        activeSession: ActiveSession.fromJson(json[r'activeSession']),
      );
    }
    return null;
  }

  static List<TabletContext> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TabletContext>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TabletContext.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TabletContext> mapFromJson(dynamic json) {
    final map = <String, TabletContext>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TabletContext.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TabletContext-objects as value to a dart map
  static Map<String, List<TabletContext>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TabletContext>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TabletContext.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'deviceId',
    'elder',
    'companionEnabled',
  };
}
