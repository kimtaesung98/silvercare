//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class EscalationsApi {
  EscalationsApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// 위급 알림 확인 (여러 번 호출해도 처음 확인한 시각·사람 유지)
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] escalationId (required):
  Future<Response> ackEscalationWithHttpInfo(
    String escalationId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/escalations/{escalationId}/ack'
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

  /// 위급 알림 확인 (여러 번 호출해도 처음 확인한 시각·사람 유지)
  ///
  /// Parameters:
  ///
  /// * [String] escalationId (required):
  Future<Escalation?> ackEscalation(
    String escalationId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await ackEscalationWithHttpInfo(
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

  /// 위급 이벤트 (FCM 알림을 눌렀을 때)
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] escalationId (required):
  Future<Response> getEscalationWithHttpInfo(
    String escalationId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/escalations/{escalationId}'
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

  /// 위급 이벤트 (FCM 알림을 눌렀을 때)
  ///
  /// Parameters:
  ///
  /// * [String] escalationId (required):
  Future<Escalation?> getEscalation(
    String escalationId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getEscalationWithHttpInfo(
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
}
