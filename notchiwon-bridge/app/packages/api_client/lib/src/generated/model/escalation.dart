//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class Escalation {
  /// Returns a new [Escalation] instance.
  Escalation({
    required this.id,
    required this.sessionId,
    required this.elder,
    required this.triggerType,
    required this.source_,
    this.utteranceText,
    this.reason,
    required this.createdAt,
    this.acknowledgedAt,
  });

  String id;

  String sessionId;

  ElderSummary elder;

  EscalationTriggerType triggerType;

  /// RULE은 규칙 필터, LLM은 Claude의 `flag_concern` 도구
  EscalationSource_Enum source_;

  /// 감지된 어르신 발화
  String? utteranceText;

  /// LLM이 감지했을 때의 설명
  String? reason;

  DateTime createdAt;

  DateTime? acknowledgedAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Escalation &&
          other.id == id &&
          other.sessionId == sessionId &&
          other.elder == elder &&
          other.triggerType == triggerType &&
          other.source_ == source_ &&
          other.utteranceText == utteranceText &&
          other.reason == reason &&
          other.createdAt == createdAt &&
          other.acknowledgedAt == acknowledgedAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (sessionId.hashCode) +
      (elder.hashCode) +
      (triggerType.hashCode) +
      (source_.hashCode) +
      (utteranceText == null ? 0 : utteranceText!.hashCode) +
      (reason == null ? 0 : reason!.hashCode) +
      (createdAt.hashCode) +
      (acknowledgedAt == null ? 0 : acknowledgedAt!.hashCode);

  @override
  String toString() =>
      'Escalation[id=$id, sessionId=$sessionId, elder=$elder, triggerType=$triggerType, source_=$source_, utteranceText=$utteranceText, reason=$reason, createdAt=$createdAt, acknowledgedAt=$acknowledgedAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'sessionId'] = this.sessionId;
    json[r'elder'] = this.elder;
    json[r'triggerType'] = this.triggerType;
    json[r'source'] = this.source_;
    if (this.utteranceText != null) {
      json[r'utteranceText'] = this.utteranceText;
    } else {
      json[r'utteranceText'] = null;
    }
    if (this.reason != null) {
      json[r'reason'] = this.reason;
    } else {
      json[r'reason'] = null;
    }
    json[r'createdAt'] = this.createdAt.toUtc().toIso8601String();
    if (this.acknowledgedAt != null) {
      json[r'acknowledgedAt'] = this.acknowledgedAt!.toUtc().toIso8601String();
    } else {
      json[r'acknowledgedAt'] = null;
    }
    return json;
  }

  /// Returns a new [Escalation] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Escalation? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Escalation[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Escalation[id]" has a null value in JSON.');
        assert(json.containsKey(r'sessionId'),
            'Required key "Escalation[sessionId]" is missing from JSON.');
        assert(json[r'sessionId'] != null,
            'Required key "Escalation[sessionId]" has a null value in JSON.');
        assert(json.containsKey(r'elder'),
            'Required key "Escalation[elder]" is missing from JSON.');
        assert(json[r'elder'] != null,
            'Required key "Escalation[elder]" has a null value in JSON.');
        assert(json.containsKey(r'triggerType'),
            'Required key "Escalation[triggerType]" is missing from JSON.');
        assert(json[r'triggerType'] != null,
            'Required key "Escalation[triggerType]" has a null value in JSON.');
        assert(json.containsKey(r'source'),
            'Required key "Escalation[source]" is missing from JSON.');
        assert(json[r'source'] != null,
            'Required key "Escalation[source]" has a null value in JSON.');
        assert(json.containsKey(r'createdAt'),
            'Required key "Escalation[createdAt]" is missing from JSON.');
        assert(json[r'createdAt'] != null,
            'Required key "Escalation[createdAt]" has a null value in JSON.');
        return true;
      }());

      return Escalation(
        id: mapValueOfType<String>(json, r'id')!,
        sessionId: mapValueOfType<String>(json, r'sessionId')!,
        elder: ElderSummary.fromJson(json[r'elder'])!,
        triggerType: EscalationTriggerType.fromJson(json[r'triggerType'])!,
        source_: EscalationSource_Enum.fromJson(json[r'source'])!,
        utteranceText: mapValueOfType<String>(json, r'utteranceText'),
        reason: mapValueOfType<String>(json, r'reason'),
        createdAt: mapDateTime(json, r'createdAt', r'')!,
        acknowledgedAt: mapDateTime(json, r'acknowledgedAt', r''),
      );
    }
    return null;
  }

  static List<Escalation> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Escalation>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Escalation.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Escalation> mapFromJson(dynamic json) {
    final map = <String, Escalation>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Escalation.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Escalation-objects as value to a dart map
  static Map<String, List<Escalation>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Escalation>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Escalation.listFromJson(
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
    'sessionId',
    'elder',
    'triggerType',
    'source',
    'createdAt',
  };
}

/// RULE은 규칙 필터, LLM은 Claude의 `flag_concern` 도구
enum EscalationSource_Enum {
  RULE._(r'RULE'),
  LLM._(r'LLM'),
  ;

  /// Instantiate a new enum with the provided value.
  const EscalationSource_Enum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [EscalationSource_Enum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static EscalationSource_Enum? fromJson(dynamic value) =>
      EscalationSource_EnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [EscalationSource_Enum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<EscalationSource_Enum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <EscalationSource_Enum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = EscalationSource_Enum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [EscalationSource_Enum] to String,
/// and [decode] dynamic data back to [EscalationSource_Enum].
class EscalationSource_EnumTypeTransformer {
  factory EscalationSource_EnumTypeTransformer() =>
      _instance ??= const EscalationSource_EnumTypeTransformer._();

  const EscalationSource_EnumTypeTransformer._();

  String encode(EscalationSource_Enum data) => data._value;

  /// Returns the instance of [EscalationSource_Enum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  EscalationSource_Enum? decode(dynamic data, {bool allowNull = true}) {
    if (data is EscalationSource_Enum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'RULE':
          return EscalationSource_Enum.RULE;
        case r'LLM':
          return EscalationSource_Enum.LLM;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static EscalationSource_EnumTypeTransformer? _instance;
}
