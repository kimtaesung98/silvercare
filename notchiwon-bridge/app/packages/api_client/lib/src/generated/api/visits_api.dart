//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class VisitsApi {
  VisitsApi([ApiClient? apiClient]) : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// 조무사 도착
  ///
  /// 방문을 완료하고, 진행 중인 세션을 `CAREGIVER_ARRIVED`로 끝냅니다. 브리핑 생성은 비동기 작업으로 이어집니다.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] visitId (required):
  Future<Response> arriveVisitWithHttpInfo(
    String visitId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/visits/{visitId}/arrive'.replaceAll('{visitId}', visitId);

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

  /// 조무사 도착
  ///
  /// 방문을 완료하고, 진행 중인 세션을 `CAREGIVER_ARRIVED`로 끝냅니다. 브리핑 생성은 비동기 작업으로 이어집니다.
  ///
  /// Parameters:
  ///
  /// * [String] visitId (required):
  Future<Visit?> arriveVisit(
    String visitId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await arriveVisitWithHttpInfo(
      visitId,
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
        'Visit',
      ) as Visit;
    }
    return null;
  }

  /// Performs an HTTP 'GET /visits/{visitId}' operation and returns the [Response].
  /// Parameters:
  ///
  /// * [String] visitId (required):
  Future<Response> getVisitWithHttpInfo(
    String visitId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/visits/{visitId}'.replaceAll('{visitId}', visitId);

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

  /// Parameters:
  ///
  /// * [String] visitId (required):
  Future<Visit?> getVisit(
    String visitId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getVisitWithHttpInfo(
      visitId,
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
        'Visit',
      ) as Visit;
    }
    return null;
  }

  /// 로그인한 조무사의 오늘 방문 목록 (예정 시각순)
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listTodayVisitsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/visits/today';

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

  /// 로그인한 조무사의 오늘 방문 목록 (예정 시각순)
  Future<VisitList?> listTodayVisits({
    Future<void>? abortTrigger,
  }) async {
    final response = await listTodayVisitsWithHttpInfo(
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
        'VisitList',
      ) as VisitList;
    }
    return null;
  }

  /// 조무사 위치 전송 (약 10초 간격)
  ///
  /// 서버가 ETA를 다시 계산하고, ETA가 기준(`SESSION_TRIGGER_ETA_MINUTES`) 이내가 되면 픽업 대기 세션을 시작합니다. 세션 시작은 조건부 `UPDATE` 한 문장으로 차지하므로 같은 방문에 위치가 동시에 여러 번 와도 세션은 하나만 생깁니다.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] visitId (required):
  ///
  /// * [LocationUpdate] locationUpdate (required):
  Future<Response> postVisitLocationWithHttpInfo(
    String visitId,
    LocationUpdate locationUpdate, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/visits/{visitId}/location'.replaceAll('{visitId}', visitId);

    // ignore: prefer_final_locals
    Object? postBody = locationUpdate;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>['application/json'];

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

  /// 조무사 위치 전송 (약 10초 간격)
  ///
  /// 서버가 ETA를 다시 계산하고, ETA가 기준(`SESSION_TRIGGER_ETA_MINUTES`) 이내가 되면 픽업 대기 세션을 시작합니다. 세션 시작은 조건부 `UPDATE` 한 문장으로 차지하므로 같은 방문에 위치가 동시에 여러 번 와도 세션은 하나만 생깁니다.
  ///
  /// Parameters:
  ///
  /// * [String] visitId (required):
  ///
  /// * [LocationUpdate] locationUpdate (required):
  Future<LocationUpdateResult?> postVisitLocation(
    String visitId,
    LocationUpdate locationUpdate, {
    Future<void>? abortTrigger,
  }) async {
    final response = await postVisitLocationWithHttpInfo(
      visitId,
      locationUpdate,
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
        'LocationUpdateResult',
      ) as LocationUpdateResult;
    }
    return null;
  }
}
