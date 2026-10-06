//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class GuardianApi {
  GuardianApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// 위급 알림 확인 (여러 번 호출해도 처음 시각 유지)
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] escalationId (required):
  Future<Response> ackGuardianEscalationWithHttpInfo(
    String escalationId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/guardian/escalations/{escalationId}/ack'
        .replaceAll('{escalationId}', escalationId);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>[];

    return apiClient.invokeAPI(
      path,
      'POST',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// 위급 알림 확인 (여러 번 호출해도 처음 시각 유지)
  ///
  /// Parameters:
  ///
  /// * [String] escalationId (required):
  Future<Escalation?> ackGuardianEscalation(
    String escalationId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await ackGuardianEscalationWithHttpInfo(
      escalationId,
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
        'Escalation',
      ) as Escalation;
    }
    return null;
  }

  /// 말동무 설정 (보호자용)
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  Future<Response> getGuardianCompanionScheduleWithHttpInfo(
    String elderId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/guardian/elders/{elderId}/companion-schedule'
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

  /// 말동무 설정 (보호자용)
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  Future<CompanionSchedule?> getGuardianCompanionSchedule(
    String elderId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getGuardianCompanionScheduleWithHttpInfo(
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

  /// 보호자와 돌보는 어르신 목록
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> getGuardianContextWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/guardian/me';

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

  /// 보호자와 돌보는 어르신 목록
  Future<GuardianContext?> getGuardianContext({
    Future<void>? abortTrigger,
  }) async {
    final response = await getGuardianContextWithHttpInfo(
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
        'GuardianContext',
      ) as GuardianContext;
    }
    return null;
  }

  /// 위급 이벤트 (알림을 눌렀을 때)
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] escalationId (required):
  Future<Response> getGuardianEscalationWithHttpInfo(
    String escalationId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/guardian/escalations/{escalationId}'
        .replaceAll('{escalationId}', escalationId);

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

  /// 위급 이벤트 (알림을 눌렀을 때)
  ///
  /// Parameters:
  ///
  /// * [String] escalationId (required):
  Future<Escalation?> getGuardianEscalation(
    String escalationId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getGuardianEscalationWithHttpInfo(
      escalationId,
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
        'Escalation',
      ) as Escalation;
    }
    return null;
  }

  /// 최근 하루 요약 목록
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  ///
  /// * [int] limit:
  ///   최대 개수 (기본 14)
  Future<Response> listDailyDigestsWithHttpInfo(
    String elderId, {
    int? limit,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/guardian/elders/{elderId}/daily-digests'
        .replaceAll('{elderId}', elderId);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (limit != null) {
      queryParams.addAll(_queryParams('', 'limit', limit));
    }

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

  /// 최근 하루 요약 목록
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  ///
  /// * [int] limit:
  ///   최대 개수 (기본 14)
  Future<DailyDigestList?> listDailyDigests(
    String elderId, {
    int? limit,
    Future<void>? abortTrigger,
  }) async {
    final response = await listDailyDigestsWithHttpInfo(
      elderId,
      limit: limit,
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
        'DailyDigestList',
      ) as DailyDigestList;
    }
    return null;
  }

  /// 확인하지 않은 위급 (최근 24시간, 말동무 대화)
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listGuardianOpenEscalationsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/guardian/escalations/open';

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

  /// 확인하지 않은 위급 (최근 24시간, 말동무 대화)
  Future<EscalationList?> listGuardianOpenEscalations({
    Future<void>? abortTrigger,
  }) async {
    final response = await listGuardianOpenEscalationsWithHttpInfo(
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
        'EscalationList',
      ) as EscalationList;
    }
    return null;
  }

  /// 말동무 설정 저장 (보호자용)
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  ///
  /// * [CompanionSchedule] companionSchedule (required):
  Future<Response> putGuardianCompanionScheduleWithHttpInfo(
    String elderId,
    CompanionSchedule companionSchedule, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/guardian/elders/{elderId}/companion-schedule'
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

  /// 말동무 설정 저장 (보호자용)
  ///
  /// Parameters:
  ///
  /// * [String] elderId (required):
  ///
  /// * [CompanionSchedule] companionSchedule (required):
  Future<CompanionSchedule?> putGuardianCompanionSchedule(
    String elderId,
    CompanionSchedule companionSchedule, {
    Future<void>? abortTrigger,
  }) async {
    final response = await putGuardianCompanionScheduleWithHttpInfo(
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

  /// 보호자 휴대폰 FCM 토큰 등록·갱신
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [FcmTokenRegistration] fcmTokenRegistration (required):
  Future<Response> putGuardianFcmTokenWithHttpInfo(
    FcmTokenRegistration fcmTokenRegistration, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/guardian/devices/me/fcm-token';

    // ignore: prefer_final_locals
    Object? postBody = fcmTokenRegistration;

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

  /// 보호자 휴대폰 FCM 토큰 등록·갱신
  ///
  /// Parameters:
  ///
  /// * [FcmTokenRegistration] fcmTokenRegistration (required):
  Future<void> putGuardianFcmToken(
    FcmTokenRegistration fcmTokenRegistration, {
    Future<void>? abortTrigger,
  }) async {
    final response = await putGuardianFcmTokenWithHttpInfo(
      fcmTokenRegistration,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }
}
