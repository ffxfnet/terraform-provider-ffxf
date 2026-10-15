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

printf '%s' "$oidc_token" |
  python3 -c '
import base64
import json
import sys

parts = sys.stdin.read().split(".")
if len(parts) != 3:
    raise SystemExit("GitHub returned a token that is not a JWT.")

payload = parts[1]
payload += "=" * (-len(payload) % 4)
claims = json.loads(base64.urlsafe_b64decode(payload))
keys = (
    "iss",
    "aud",
    "sub",
    "repository",
    "repository_id",
    "repository_owner",
    "repository_owner_id",
    "ref",
    "event_name",
    "workflow_ref",
    "job_workflow_ref",
)
print("OIDC token identity claims (decoded locally; signature is verified by the API):")
print(json.dumps({key: claims[key] for key in keys if key in claims}, indent=2))
'

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
