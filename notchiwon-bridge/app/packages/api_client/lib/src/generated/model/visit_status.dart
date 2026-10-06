//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

enum VisitStatus {
  SCHEDULED._(r'SCHEDULED'),
  EN_ROUTE._(r'EN_ROUTE'),
  SESSION_ACTIVE._(r'SESSION_ACTIVE'),
  COMPLETED._(r'COMPLETED'),
  CANCELLED._(r'CANCELLED'),
  ;

  /// Instantiate a new enum with the provided value.
  const VisitStatus._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [VisitStatus] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static VisitStatus? fromJson(dynamic value) =>
      VisitStatusTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [VisitStatus]
  /// that were successfully decoded from the passed [JSON][json].
  static List<VisitStatus> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <VisitStatus>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = VisitStatus.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [VisitStatus] to String,
/// and [decode] dynamic data back to [VisitStatus].
class VisitStatusTypeTransformer {
  factory VisitStatusTypeTransformer() =>
      _instance ??= const VisitStatusTypeTransformer._();

  const VisitStatusTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(VisitStatus data) => data._value;

  /// Returns the instance of [VisitStatus] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  VisitStatus? decode(dynamic data, {bool allowNull = true}) {
    if (data is VisitStatus) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'SCHEDULED':
          return VisitStatus.SCHEDULED;
        case r'EN_ROUTE':
          return VisitStatus.EN_ROUTE;
        case r'SESSION_ACTIVE':
          return VisitStatus.SESSION_ACTIVE;
        case r'COMPLETED':
          return VisitStatus.COMPLETED;
        case r'CANCELLED':
          return VisitStatus.CANCELLED;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static VisitStatusTypeTransformer? _instance;
}
