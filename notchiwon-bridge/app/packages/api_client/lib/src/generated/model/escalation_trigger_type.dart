//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

enum EscalationTriggerType {
  FALL_MENTION._(r'FALL_MENTION'),
  PAIN_COMPLAINT._(r'PAIN_COMPLAINT'),
  SELF_OR_OTHER_HARM._(r'SELF_OR_OTHER_HARM'),
  OTHER_ANOMALY._(r'OTHER_ANOMALY'),
  ;

  /// Instantiate a new enum with the provided value.
  const EscalationTriggerType._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [EscalationTriggerType] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static EscalationTriggerType? fromJson(dynamic value) =>
      EscalationTriggerTypeTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [EscalationTriggerType]
  /// that were successfully decoded from the passed [JSON][json].
  static List<EscalationTriggerType> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <EscalationTriggerType>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = EscalationTriggerType.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [EscalationTriggerType] to String,
/// and [decode] dynamic data back to [EscalationTriggerType].
class EscalationTriggerTypeTypeTransformer {
  factory EscalationTriggerTypeTypeTransformer() =>
      _instance ??= const EscalationTriggerTypeTypeTransformer._();

  const EscalationTriggerTypeTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(EscalationTriggerType data) => data._value;

  /// Returns the instance of [EscalationTriggerType] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  EscalationTriggerType? decode(dynamic data, {bool allowNull = true}) {
    if (data is EscalationTriggerType) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'FALL_MENTION':
          return EscalationTriggerType.FALL_MENTION;
        case r'PAIN_COMPLAINT':
          return EscalationTriggerType.PAIN_COMPLAINT;
        case r'SELF_OR_OTHER_HARM':
          return EscalationTriggerType.SELF_OR_OTHER_HARM;
        case r'OTHER_ANOMALY':
          return EscalationTriggerType.OTHER_ANOMALY;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static EscalationTriggerTypeTypeTransformer? _instance;
}
