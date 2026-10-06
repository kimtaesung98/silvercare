//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

library openapi.api;

import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:collection/collection.dart';
import 'package:http/http.dart';
import 'package:intl/intl.dart';
import 'package:meta/meta.dart';

part 'api_client.dart';
part 'api_helper.dart';
part 'api_exception.dart';
part 'auth/authentication.dart';
part 'auth/api_key_auth.dart';
part 'auth/oauth.dart';
part 'auth/http_basic_auth.dart';
part 'auth/http_bearer_auth.dart';

part 'api/auth_api.dart';
part 'api/briefings_api.dart';
part 'api/companion_api.dart';
part 'api/devices_api.dart';
part 'api/escalations_api.dart';
part 'api/system_api.dart';
part 'api/tablet_api.dart';
part 'api/visits_api.dart';

part 'model/active_session.dart';
part 'model/api_error.dart';
part 'model/briefing.dart';
part 'model/caregiver.dart';
part 'model/caregiver_login_request.dart';
part 'model/caregiver_login_response.dart';
part 'model/companion_schedule.dart';
part 'model/daily_digest.dart';
part 'model/elder_summary.dart';
part 'model/emotion_tag.dart';
part 'model/escalation.dart';
part 'model/escalation_list.dart';
part 'model/escalation_trigger_type.dart';
part 'model/fcm_token_registration.dart';
part 'model/health.dart';
part 'model/location_update.dart';
part 'model/location_update_result.dart';
part 'model/opener_category.dart';
part 'model/opener_clip.dart';
part 'model/opener_clip_manifest.dart';
part 'model/place.dart';
part 'model/session_mode.dart';
part 'model/tablet_context.dart';
part 'model/visit.dart';
part 'model/visit_list.dart';
part 'model/visit_status.dart';

/// An [ApiClient] instance that uses the default values obtained from
/// the OpenAPI specification file.
var defaultApiClient = ApiClient();

const _delimiters = {'csv': ',', 'ssv': ' ', 'tsv': '\t', 'pipes': '|'};
const _dateEpochMarker = 'epoch';
const _deepEquality = DeepCollectionEquality();
final _dateFormatter = DateFormat('yyyy-MM-dd');
final _regList = RegExp(r'^List<(.*)>$');
final _regSet = RegExp(r'^Set<(.*)>$');
final _regMap = RegExp(r'^Map<String,(.*)>$');

bool _isEpochMarker(String? pattern) =>
    pattern == _dateEpochMarker || pattern == '/$_dateEpochMarker/';
