//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

enum SessionMode {
  PICKUP_BRIDGE._(r'PICKUP_BRIDGE'),
  COMPANION._(r'COMPANION'),
  ;

  /// Instantiate a new enum with the provided value.
  const SessionMode._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SessionMode] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SessionMode? fromJson(dynamic value) =>
      SessionModeTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SessionMode]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SessionMode> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SessionMode>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SessionMode.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SessionMode] to String,
/// and [decode] dynamic data back to [SessionMode].
class SessionModeTypeTransformer {
  factory SessionModeTypeTransformer() =>
      _instance ??= const SessionModeTypeTransformer._();

  const SessionModeTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(SessionMode data) => data._value;

  /// Returns the instance of [SessionMode] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SessionMode? decode(dynamic data, {bool allowNull = true}) {
    if (data is SessionMode) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'PICKUP_BRIDGE':
          return SessionMode.PICKUP_BRIDGE;
        case r'COMPANION':
          return SessionMode.COMPANION;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SessionModeTypeTransformer? _instance;
}
