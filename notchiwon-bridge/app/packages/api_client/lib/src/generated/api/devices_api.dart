//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class DevicesApi {
  DevicesApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// 조무사 휴대폰 FCM 토큰 등록·갱신
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [FcmTokenRegistration] fcmTokenRegistration (required):
  Future<Response> putCaregiverFcmTokenWithHttpInfo(
    FcmTokenRegistration fcmTokenRegistration, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/devices/me/fcm-token';

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

  /// 조무사 휴대폰 FCM 토큰 등록·갱신
  ///
  /// Parameters:
  ///
  /// * [FcmTokenRegistration] fcmTokenRegistration (required):
  Future<void> putCaregiverFcmToken(
    FcmTokenRegistration fcmTokenRegistration, {
    Future<void>? abortTrigger,
  }) async {
    final response = await putCaregiverFcmTokenWithHttpInfo(
      fcmTokenRegistration,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }
}
