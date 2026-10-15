#!/usr/bin/env bash
set -euo pipefail

: "${ACTIONS_ID_TOKEN_REQUEST_URL:?GitHub Actions OIDC request URL is missing}"
: "${ACTIONS_ID_TOKEN_REQUEST_TOKEN:?GitHub Actions OIDC request token is missing}"
: "${OIDC_AUDIENCE:?OIDC_AUDIENCE is required}"

endpoint="${FFXF_ENDPOINT:-https://api.ffxf.net/v1}"
response_file="$(mktemp)"
trap 'rm -f "$response_file"' EXIT

oidc_token="$(
  curl --fail --silent --show-error --get \
    --data-urlencode "audience=$OIDC_AUDIENCE" \
    -H "Authorization: Bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
    "$ACTIONS_ID_TOKEN_REQUEST_URL" |
    jq --exit-status --raw-output '.value | strings | select(length > 0)'
)"
echo "::add-mask::$oidc_token"

http_status="$(
  curl --silent --show-error \
    --output "$response_file" \
    --write-out '%{http_code}' \
    --header "Authorization: Bearer $oidc_token" \
    --header "Accept: application/json" \
    "${endpoint%/}/vms"
)"

if [[ "$http_status" != 2* ]]; then
  printf 'OIDC smoke test failed: GET %s/vms returned HTTP %s\n' "${endpoint%/}" "$http_status" >&2
  if [[ -s "$response_file" ]]; then
    cat "$response_file" >&2
    printf '\n' >&2
  fi
  exit 1
fi

printf 'OIDC authentication succeeded: GET %s/vms returned HTTP %s\n' "${endpoint%/}" "$http_status"
