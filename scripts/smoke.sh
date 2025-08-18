#!/usr/bin/env bash
set -euo pipefail

URL="${1:-http://localhost:8080/health}"
ATTEMPTS=${ATTEMPTS:-20}
SLEEP=${SLEEP:-2}

for i in $(seq 1 "$ATTEMPTS"); do
  code=$(curl -s -o /dev/null -w "%{http_code}" "$URL" || true)
  if [ "$code" = "200" ]; then
    echo "Smoke OK: $URL"
    exit 0
  fi
  sleep "$SLEEP"

done

echo "Smoke FAILED: $URL"
exit 1

