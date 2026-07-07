#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

export ANISETTE_URL="${ANISETTE_URL:-http://localhost:6969}"

PRIVATE_KEY="${KNIRV_IOS_P12:-$ROOT_DIR/.config/anisette/private_key.p12}"
KEY_PASSWORD="${KNIRV_IOS_P12_PASSWORD:-our_p12_password}"
PROVISION_PROFILE="${KNIRV_IOS_PROVISION_PROFILE:-$ROOT_DIR/.config/anisette/free_profile.mobileprovision}"

UNSIGNED_IPA="$ROOT_DIR/build/knirvconroller-ios-unsigned.ipa"
OUTPUT_IPA="$ROOT_DIR/build/knirvconroller-ios-release.ipa"

missing=0

if ! command -v zsign >/dev/null 2>&1; then
  echo "Error: zsign is not installed or is not on PATH."
  missing=1
fi

if [ ! -f "$PRIVATE_KEY" ]; then
  echo "Error: missing private key: $PRIVATE_KEY"
  missing=1
fi

if [ ! -f "$PROVISION_PROFILE" ]; then
  echo "Error: missing provisioning profile: $PROVISION_PROFILE"
  missing=1
fi

if [ ! -f "$UNSIGNED_IPA" ]; then
  echo "Error: missing unsigned IPA: $UNSIGNED_IPA"
  missing=1
fi

if [ "$missing" -ne 0 ]; then
  echo "Signing inputs are not complete yet."
  exit 1
fi

echo "Using Anisette URL: $ANISETTE_URL"
echo "Signing $UNSIGNED_IPA"

zsign -k "$PRIVATE_KEY" \
  -p "$KEY_PASSWORD" \
  -m "$PROVISION_PROFILE" \
  -o "$OUTPUT_IPA" \
  "$UNSIGNED_IPA"

echo "Signed IPA ready: $OUTPUT_IPA"
