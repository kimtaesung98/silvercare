#!/usr/bin/env bash
# 로컬 개발 도우미. notchiwon-bridge/.env 하나만 채우면 서버·관리 명령·앱 실행이
# 모두 그 값을 읽습니다. macOS 기본 bash(3.2)에서도 돌도록 씁니다.
#
#   scripts/dev.sh check       빠진 값 확인
#   scripts/dev.sh infra       postgres, redis 띄우기 (Docker)
#   scripts/dev.sh server      서버 실행 (make run)
#   scripts/dev.sh seed        데모 데이터 만들고 토큰·로그인을 .env에 저장
#   scripts/dev.sh openers     0번 문장 음성 합성 (Clova Voice나 Amazon Polly 필요)
#   scripts/dev.sh cli         텍스트로 어르신 대화 (elder-cli)
#   scripts/dev.sh devices     연결된 기기·에뮬레이터 목록
#   scripts/dev.sh tablet      어르신 태블릿 앱 실행
#   scripts/dev.sh caregiver   조무사 앱 실행
#   scripts/dev.sh guardian    보호자 앱 실행
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT/.env"

die() { echo "dev.sh: $*" >&2; exit 1; }

load_env() {
  [ -f "$ENV_FILE" ] || die ".env가 없습니다. 먼저: cp $ROOT/.env.example $ENV_FILE"
  set -a
  # shellcheck disable=SC1090
  . "$ENV_FILE"
  set +a
}

# set_env KEY VALUE: .env의 KEY 줄을 바꾸고, 없으면 끝에 붙입니다.
set_env() {
  local key="$1" value="$2" tmp
  tmp="$(mktemp)"
  if grep -q "^${key}=" "$ENV_FILE"; then
    awk -v k="$key" -v v="$value" 'index($0, k "=") == 1 { print k "=\"" v "\""; next } { print }' "$ENV_FILE" >"$tmp"
  else
    cat "$ENV_FILE" >"$tmp"
    printf '%s="%s"\n' "$key" "$value" >>"$tmp"
  fi
  cat "$tmp" >"$ENV_FILE"
  rm -f "$tmp"
}

# report 묶음이름 변수이름...: 각 값이 채워졌는지 찍습니다.
report() {
  local group="$1" name
  shift
  echo "[$group]"
  for name in "$@"; do
    if [ -n "${!name:-}" ]; then
      echo "  ok    $name"
    else
      echo "  비어있음 $name"
    fi
  done
}

# 앱 실행용 값. 예전에 만든 .env에 줄이 없으면 check가 빈 줄을 붙여 둡니다.
APP_KEYS="APP_SERVER_URL DEVICE_TOKEN CAREGIVER_LOGIN CAREGIVER_PASSWORD GUARDIAN_LOGIN GUARDIAN_PASSWORD
  FIREBASE_API_KEY FIREBASE_SENDER_ID FIREBASE_PROJECT_ID CAREGIVER_FIREBASE_APP_ID GUARDIAN_FIREBASE_APP_ID
  TABLET_DEVICE CAREGIVER_DEVICE GUARDIAN_DEVICE TABLET_SPEECH"

cmd_check() {
  load_env
  local key added=""
  for key in $APP_KEYS; do
    if ! grep -q "^${key}=" "$ENV_FILE"; then
      if [ "$key" = APP_SERVER_URL ]; then set_env "$key" "http://10.0.2.2:8000"; else set_env "$key" ""; fi
      added="$added $key"
    fi
  done
  [ -n "$added" ] && echo ".env에 빈 줄을 추가했습니다:$added" && echo
  load_env
  report "필수" DATABASE_URL AUTH_SECRET
  report "Claude (없으면 대체 문장만)" ANTHROPIC_API_KEY
  report "Clova 음성 인식 (없으면 태블릿 내장 인식)" NAVER_CLOVA_CLIENT_ID NAVER_CLOVA_CLIENT_SECRET
  if [ "${TTS_PROVIDER:-clova}" = polly ]; then
    report "Amazon Polly 목소리" AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY
  fi
  report "카카오 ETA (없으면 예정 시각 기준)" KAKAO_MOBILITY_API_KEY
  report "서버 푸시 (없으면 앱 목록에만)" FCM_CREDENTIALS_FILE
  report "앱 공통" APP_SERVER_URL
  report "태블릿 (seed가 채움)" DEVICE_TOKEN
  report "앱 Firebase (없으면 푸시 없이)" FIREBASE_API_KEY FIREBASE_SENDER_ID FIREBASE_PROJECT_ID \
    CAREGIVER_FIREBASE_APP_ID GUARDIAN_FIREBASE_APP_ID
  report "기기 id (없으면 flutter가 물음)" TABLET_DEVICE CAREGIVER_DEVICE GUARDIAN_DEVICE
  echo "[태블릿 음성]"
  echo "  어르신 말 알아듣기: $(tablet_speech) (TABLET_SPEECH로 바꿀 수 있음)"
  echo "  AI 목소리: $(ai_voice)"
  if [ -n "${AUTH_SECRET:-}" ] && [ "${#AUTH_SECRET}" -lt 32 ]; then
    echo "주의: AUTH_SECRET이 32자보다 짧아 서버가 시작하지 않습니다 (openssl rand -base64 48)"
  fi
  if [ -n "${FCM_CREDENTIALS_FILE:-}" ] && [ ! -f "$FCM_CREDENTIALS_FILE" ]; then
    echo "주의: FCM_CREDENTIALS_FILE 경로에 파일이 없습니다: $FCM_CREDENTIALS_FILE"
  fi
}

cmd_infra() {
  cd "$ROOT"
  docker compose up -d postgres redis
}

cmd_server() {
  load_env
  cd "$ROOT/server"
  exec make run
}

cmd_seed() {
  load_env
  cd "$ROOT/server"
  local out
  out="$(go run ./cmd/admin seed-demo)"
  echo "$out"
  field() { echo "$out" | sed -n "s/^$1: *//p"; }
  local caregiver guardian token
  caregiver="$(field 'caregiver login')"
  guardian="$(field 'guardian login')"
  token="$(field 'tablet token')"
  [ -n "$token" ] || die "seed-demo 출력에서 태블릿 토큰을 찾지 못했습니다"
  set_env DEVICE_TOKEN "$token"
  set_env CAREGIVER_LOGIN "${caregiver%% / *}"
  set_env CAREGIVER_PASSWORD "${caregiver##* / }"
  set_env GUARDIAN_LOGIN "${guardian%% / *}"
  set_env GUARDIAN_PASSWORD "${guardian##* / }"
  echo
  echo ".env에 DEVICE_TOKEN과 로그인 정보를 저장했습니다. 태블릿 앱은 다시 실행해야 새 토큰을 씁니다."
}

# ai_voice: 서버가 AI 문장을 어느 목소리로 만드는지 (TTS_PROVIDER와 키로 정해짐).
ai_voice() {
  case "${TTS_PROVIDER:-clova}" in
    polly)
      if [ -n "${AWS_ACCESS_KEY_ID:-}" ] && [ -n "${AWS_SECRET_ACCESS_KEY:-}" ]; then
        echo "Amazon Polly (${POLLY_VOICE:-Seoyeon}, ${POLLY_ENGINE:-neural})"
      else
        echo "Amazon Polly인데 AWS 키가 비어 있음 (서버가 문장마다 실패하고 태블릿 내장 TTS로 읽음)"
      fi
      ;;
    clova)
      if [ -n "${NAVER_CLOVA_CLIENT_ID:-}" ] && [ -n "${NAVER_CLOVA_CLIENT_SECRET:-}" ]; then
        echo "Clova Voice (Application에 CLOVA Voice - Premium이 있어야 함)"
      else
        echo "태블릿 내장 TTS"
      fi
      ;;
    *) echo "TTS_PROVIDER가 잘못됨: ${TTS_PROVIDER} (clova 또는 polly, 서버가 시작하지 않음)" ;;
  esac
}

cmd_openers() {
  load_env
  cd "$ROOT/server"
  go run ./cmd/admin synth-openers
}

cmd_cli() {
  load_env
  [ -n "${DEVICE_TOKEN:-}" ] || die "DEVICE_TOKEN이 비어 있습니다. 먼저 scripts/dev.sh seed"
  cd "$ROOT/server"
  exec go run ./cmd/elder-cli -server "http://localhost:${PORT:-8000}" -token "$DEVICE_TOKEN"
}

cmd_devices() {
  flutter devices
}

# run_app 폴더 기기id [--dart-define=...]...
run_app() {
  local dir="$1" device="$2"
  shift 2
  local args=(run)
  [ -n "$device" ] && args+=(-d "$device")
  args+=("--dart-define=SERVER_URL=${APP_SERVER_URL:-http://10.0.2.2:8000}" "$@")
  cd "$ROOT/app/apps/$dir"
  exec flutter "${args[@]}"
}

# firebase_defines 앱ID: 네 값이 다 있을 때만 Firebase define을 찍습니다.
firebase_defines() {
  if [ -n "${FIREBASE_API_KEY:-}" ] && [ -n "$1" ] && [ -n "${FIREBASE_SENDER_ID:-}" ] && [ -n "${FIREBASE_PROJECT_ID:-}" ]; then
    printf '%s\n' "--dart-define=FIREBASE_API_KEY=$FIREBASE_API_KEY" "--dart-define=FIREBASE_APP_ID=$1" \
      "--dart-define=FIREBASE_SENDER_ID=$FIREBASE_SENDER_ID" "--dart-define=FIREBASE_PROJECT_ID=$FIREBASE_PROJECT_ID"
  else
    echo "dev.sh: Firebase 값이 비어 있어 푸시 없이 실행합니다" >&2
  fi
}

# tablet_speech: 태블릿이 어르신 말을 알아듣는 방식. TABLET_SPEECH가 비면
# Clova 키가 있을 때 server(서버가 Clova로), 없을 때 device(태블릿 내장 음성 인식).
tablet_speech() {
  if [ -n "${TABLET_SPEECH:-}" ]; then
    echo "$TABLET_SPEECH"
  elif [ -n "${NAVER_CLOVA_CLIENT_ID:-}" ] && [ -n "${NAVER_CLOVA_CLIENT_SECRET:-}" ]; then
    echo server
  else
    echo device
  fi
}

cmd_tablet() {
  load_env
  [ -n "${DEVICE_TOKEN:-}" ] || die "DEVICE_TOKEN이 비어 있습니다. 먼저 scripts/dev.sh seed"
  local speech
  speech="$(tablet_speech)"
  case "$speech" in
    device | server) ;;
    *) die "TABLET_SPEECH는 device나 server여야 합니다: $speech" ;;
  esac
  echo "dev.sh: 어르신 말 알아듣기 = $speech"
  run_app elder_tablet "${TABLET_DEVICE:-}" "--dart-define=DEVICE_TOKEN=$DEVICE_TOKEN" "--dart-define=SPEECH=$speech"
}

cmd_phone_app() {
  local dir="$1" device="$2" app_id="$3" login="$4" password="$5" defines=() line
  while IFS= read -r line; do
    [ -n "$line" ] && defines+=("$line")
  done < <(firebase_defines "$app_id")
  [ -n "$login" ] && echo "dev.sh: 로그인 정보 $login / $password"
  run_app "$dir" "$device" ${defines[@]+"${defines[@]}"}
}

cmd_caregiver() {
  load_env
  cmd_phone_app caregiver "${CAREGIVER_DEVICE:-}" "${CAREGIVER_FIREBASE_APP_ID:-}" \
    "${CAREGIVER_LOGIN:-}" "${CAREGIVER_PASSWORD:-}"
}

cmd_guardian() {
  load_env
  cmd_phone_app guardian "${GUARDIAN_DEVICE:-}" "${GUARDIAN_FIREBASE_APP_ID:-}" \
    "${GUARDIAN_LOGIN:-}" "${GUARDIAN_PASSWORD:-}"
}

usage() {
  sed -n '2,15p' "$0" | sed 's/^# \{0,1\}//'
  exit 1
}

case "${1:-}" in
  check) cmd_check ;;
  infra) cmd_infra ;;
  server) cmd_server ;;
  seed) cmd_seed ;;
  openers) cmd_openers ;;
  cli) cmd_cli ;;
  devices) cmd_devices ;;
  tablet) cmd_tablet ;;
  caregiver) cmd_caregiver ;;
  guardian) cmd_guardian ;;
  *) usage ;;
esac
