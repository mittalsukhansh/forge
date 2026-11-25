#!/bin/bash
set -euo pipefail

IMAGE="${1:?Image name required}"

mkdir -p reports

echo "========== TRIVY SECURITY SCAN =========="

docker pull aquasec/trivy:latest
docker volume create forge-trivy-cache

scan() {
    docker run --rm \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v forge-trivy-cache:/root/.cache/trivy \
        aquasec/trivy:latest image \
        --image-src docker \
        --scanners vuln \
        --timeout 10m \
        "$@" "$IMAGE"
}

scan --severity HIGH,CRITICAL \
    --exit-code 0 | tee reports/vulnerabilities.txt

scan --severity CRITICAL \
    --ignore-unfixed \
    --exit-code 1

echo "SECURITY_GATE_PASSED"
