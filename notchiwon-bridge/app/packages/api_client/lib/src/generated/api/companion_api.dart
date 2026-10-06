//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class CompanionApi {
  CompanionApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// 말동무 설정 (안부 시각, 취침 시간, 하루 토큰 한도)
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  Future<Response> getCompanionScheduleWithHttpInfo(
    String elderId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/elders/{elderId}/companion-schedule'
        .replaceAll('{elderId}', elderId);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>[];

    return apiClient.invokeAPI(
      path,
      'GET',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// 말동무 설정 (안부 시각, 취침 시간, 하루 토큰 한도)
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  Future<CompanionSchedule?> getCompanionSchedule(
    String elderId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getCompanionScheduleWithHttpInfo(
      elderId,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      return await apiClient.deserializeAsync(
        await _decodeBodyBytes(response),
        'CompanionSchedule',
      ) as CompanionSchedule;
    }
    return null;
  }

  /// 보호자용 하루 요약
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  ///
  /// * [DateTime] date (required):
  ///   한국 시각 기준 날짜
  Future<Response> getDailyDigestWithHttpInfo(
    String elderId,
    DateTime date, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/elders/{elderId}/daily-digests/{date}'
        .replaceAll('{elderId}', elderId)
        .replaceAll('{date}', date.toString());

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>[];

    return apiClient.invokeAPI(
      path,
      'GET',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// 보호자용 하루 요약
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  ///
  /// * [DateTime] date (required):
  ///   한국 시각 기준 날짜
  Future<DailyDigest?> getDailyDigest(
    String elderId,
    DateTime date, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getDailyDigestWithHttpInfo(
      elderId,
      date,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      return await apiClient.deserializeAsync(
        await _decodeBodyBytes(response),
        'DailyDigest',
      ) as DailyDigest;
    }
    return null;
  }

  /// 말동무 설정 저장
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  ///
  /// * [CompanionSchedule] companionSchedule (required):
  Future<Response> putCompanionScheduleWithHttpInfo(
    String elderId,
    CompanionSchedule companionSchedule, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/elders/{elderId}/companion-schedule'
        .replaceAll('{elderId}', elderId);

    // ignore: prefer_final_locals
    Object? postBody = companionSchedule;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>['application/json'];

    return apiClient.invokeAPI(
      path,
      'PUT',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// 말동무 설정 저장
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  ///
  /// * [CompanionSchedule] companionSchedule (required):
  Future<CompanionSchedule?> putCompanionSchedule(
    String elderId,
    CompanionSchedule companionSchedule, {
    Future<void>? abortTrigger,
  }) async {
    final response = await putCompanionScheduleWithHttpInfo(
      elderId,
      companionSchedule,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      return await apiClient.deserializeAsync(
        await _decodeBodyBytes(response),
        'CompanionSchedule',
      ) as CompanionSchedule;
    }
    return null;
  }
}
