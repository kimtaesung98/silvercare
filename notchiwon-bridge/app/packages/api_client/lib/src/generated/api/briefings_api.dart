//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class BriefingsApi {
  BriefingsApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// 도착 브리핑 카드
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] sessionId (required):
  Future<Response> getSessionBriefingWithHttpInfo(
    String sessionId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path =
        r'/sessions/{sessionId}/briefing'.replaceAll('{sessionId}', sessionId);

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

  /// 도착 브리핑 카드
  ///
  /// Parameters:
  ///
  /// * [String] sessionId (required):
  Future<Briefing?> getSessionBriefing(
    String sessionId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getSessionBriefingWithHttpInfo(
      sessionId,
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
        'Briefing',
      ) as Briefing;
    }
    return null;
  }

  /// 브리핑 읽음 표시 (여러 번 호출해도 처음 시각 유지)
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] sessionId (required):
  Future<Response> markBriefingReadWithHttpInfo(
    String sessionId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/sessions/{sessionId}/briefing/read'
        .replaceAll('{sessionId}', sessionId);

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

  /// 브리핑 읽음 표시 (여러 번 호출해도 처음 시각 유지)
  ///
  /// Parameters:
  ///
  /// * [String] sessionId (required):
  Future<Briefing?> markBriefingRead(
    String sessionId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await markBriefingReadWithHttpInfo(
      sessionId,
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
        'Briefing',
      ) as Briefing;
    }
    return null;
  }
}
