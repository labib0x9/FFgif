#!/usr/bin/env bash
set -eo pipefail

# Resolve repository root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Ensure reports directory exists
mkdir -p "${REPO_ROOT}/reports"

echo "==> Starting OWASP ZAP API Scan for FFgif..."
echo "==> OpenAPI spec: ${REPO_ROOT}/docs/openapi.yaml"
echo "==> Target network: ffgif_backend (http://api:8080)"
echo "==> Reports destination: ${REPO_ROOT}/reports"

# Run OWASP ZAP API scan inside Docker
# Notes:
#  -m 3: Wait up to 3 minutes for the ZAP Java daemon to initialize
#  -v "${REPO_ROOT}:/zap/wrk:rw": Map repository root to container working directory
#  -t: Allocate TTY for proper terminal output
set +e
docker run --rm -t \
  --network ffgif_backend \
  -v "${REPO_ROOT}:/zap/wrk:rw" \
  ghcr.io/zaproxy/zaproxy:stable \
  zap-api-scan.py \
  -t docs/openapi.yaml \
  -f openapi \
  -m 3 \
  -r reports/zap-report.html \
  -J reports/zap-report.json \
  "$@"

SCAN_STATUS=$?
set -e

echo ""
echo "--------------------------------------------------------"
if [ ${SCAN_STATUS} -eq 0 ]; then
  echo "==> [PASS] ZAP scan completed with 0 warnings/failures."
elif [ ${SCAN_STATUS} -eq 2 ]; then
  echo "==> [WARN] ZAP scan completed: warnings/advisories were identified."
elif [ ${SCAN_STATUS} -eq 1 ]; then
  echo "==> [FAIL] ZAP scan completed: high/critical issues found."
else
  echo "==> [ERROR] ZAP scan exited with status code ${SCAN_STATUS}."
fi
echo "==> HTML Report: ${REPO_ROOT}/reports/zap-report.html"
echo "==> JSON Report: ${REPO_ROOT}/reports/zap-report.json"
echo "--------------------------------------------------------"

exit ${SCAN_STATUS}
