//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

/// RECALL 회상·이야기, QUESTION 질문, EMOTION 감정 표현, SHORT_ANSWER 짧은 대답, GREETING 인사, ESCALATION 위급 감지(Claude 호출 없음), FILLER 1번 문장이 늦을 때 트는 추임새
enum OpenerCategory {
  RECALL._(r'RECALL'),
  QUESTION._(r'QUESTION'),
  EMOTION._(r'EMOTION'),
  SHORT_ANSWER._(r'SHORT_ANSWER'),
  GREETING._(r'GREETING'),
  ESCALATION._(r'ESCALATION'),
  FILLER._(r'FILLER'),
  ;

  /// Instantiate a new enum with the provided value.
  const OpenerCategory._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [OpenerCategory] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static OpenerCategory? fromJson(dynamic value) =>
      OpenerCategoryTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [OpenerCategory]
  /// that were successfully decoded from the passed [JSON][json].
  static List<OpenerCategory> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <OpenerCategory>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = OpenerCategory.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [OpenerCategory] to String,
/// and [decode] dynamic data back to [OpenerCategory].
class OpenerCategoryTypeTransformer {
  factory OpenerCategoryTypeTransformer() =>
      _instance ??= const OpenerCategoryTypeTransformer._();

  const OpenerCategoryTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(OpenerCategory data) => data._value;

  /// Returns the instance of [OpenerCategory] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  OpenerCategory? decode(dynamic data, {bool allowNull = true}) {
    if (data is OpenerCategory) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'RECALL':
          return OpenerCategory.RECALL;
        case r'QUESTION':
          return OpenerCategory.QUESTION;
        case r'EMOTION':
          return OpenerCategory.EMOTION;
        case r'SHORT_ANSWER':
          return OpenerCategory.SHORT_ANSWER;
        case r'GREETING':
          return OpenerCategory.GREETING;
        case r'ESCALATION':
          return OpenerCategory.ESCALATION;
        case r'FILLER':
          return OpenerCategory.FILLER;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static OpenerCategoryTypeTransformer? _instance;
}
