#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "" ] || [ "${2:-}" = "" ]; then
  echo "Usage: $0 <threshold-percent> <coverprofile>"
  exit 2
fi

THRESHOLD="$1"
PROFILE="$2"

if [ ! -f "$PROFILE" ]; then
  echo "Coverage profile not found: $PROFILE"
  exit 2
fi

TMP_OUT=$(mktemp)
go tool cover -func="$PROFILE" | tee "$TMP_OUT"
TOTAL=$(awk '/total:/ {gsub("%", "", $3); print $3}' "$TMP_OUT")
rm -f "$TMP_OUT"

awk -v t="$THRESHOLD" -v a="$TOTAL" 'BEGIN { if (a+0 < t+0) { exit 1 } }' || {
  echo "Coverage ${TOTAL}% is below threshold ${THRESHOLD}%"
  exit 1
}

echo "Coverage ${TOTAL}% meets threshold ${THRESHOLD}%"

