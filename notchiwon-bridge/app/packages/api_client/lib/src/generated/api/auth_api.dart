//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class AuthApi {
  AuthApi([ApiClient? apiClient]) : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// 조무사 임시 로그인
  ///
  /// 센터가 발급한 아이디·비밀번호로 로그인합니다. 로그인 방식이 정해지면 바뀔 수 있습니다.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [CaregiverLoginRequest] caregiverLoginRequest (required):
  Future<Response> loginCaregiverWithHttpInfo(
    CaregiverLoginRequest caregiverLoginRequest, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/auth/caregiver/login';

    // ignore: prefer_final_locals
    Object? postBody = caregiverLoginRequest;

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

  /// 조무사 임시 로그인
  ///
  /// 센터가 발급한 아이디·비밀번호로 로그인합니다. 로그인 방식이 정해지면 바뀔 수 있습니다.
  ///
  /// Parameters:
  ///
  /// * [CaregiverLoginRequest] caregiverLoginRequest (required):
  Future<CaregiverLoginResponse?> loginCaregiver(
    CaregiverLoginRequest caregiverLoginRequest, {
    Future<void>? abortTrigger,
  }) async {
    final response = await loginCaregiverWithHttpInfo(
      caregiverLoginRequest,
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
        'CaregiverLoginResponse',
      ) as CaregiverLoginResponse;
    }
    return null;
  }
}
