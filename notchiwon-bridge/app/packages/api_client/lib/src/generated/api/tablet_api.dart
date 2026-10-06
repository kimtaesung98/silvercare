//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of openapi.api;

class TabletApi {
  TabletApi([ApiClient? apiClient]) : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// 0번 문장 음성 파일
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] clipId (required):
  Future<Response> getOpenerClipAudioWithHttpInfo(
    String clipId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path =
        r'/tablet/opener-clips/{clipId}/audio'.replaceAll('{clipId}', clipId);

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

  /// 0번 문장 음성 파일
  ///
  /// Parameters:
  ///
  /// * [String] clipId (required):
  Future<MultipartFile?> getOpenerClipAudio(
    String clipId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getOpenerClipAudioWithHttpInfo(
      clipId,
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
        'MultipartFile',
      ) as MultipartFile;
    }
    return null;
  }

  /// 태블릿이 켜질 때 받는 기기·어르신 정보
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> getTabletContextWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/tablet/me';

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

  /// 태블릿이 켜질 때 받는 기기·어르신 정보
  Future<TabletContext?> getTabletContext({
    Future<void>? abortTrigger,
  }) async {
    final response = await getTabletContextWithHttpInfo(
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
        'TabletContext',
      ) as TabletContext;
    }
    return null;
  }

  /// 0번 문장 목록
  ///
  /// 어르신 목소리(`preferred_tts_voice`)로 미리 합성한 0번 문장 목록입니다. 태블릿은 `version`이 바뀌었을 때만 음성을 다시 내려받아 캐시합니다.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listOpenerClipsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/tablet/opener-clips';

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

  /// 0번 문장 목록
  ///
  /// 어르신 목소리(`preferred_tts_voice`)로 미리 합성한 0번 문장 목록입니다. 태블릿은 `version`이 바뀌었을 때만 음성을 다시 내려받아 캐시합니다.
  Future<OpenerClipManifest?> listOpenerClips({
    Future<void>? abortTrigger,
  }) async {
    final response = await listOpenerClipsWithHttpInfo(
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
        'OpenerClipManifest',
      ) as OpenerClipManifest;
    }
    return null;
  }
}
