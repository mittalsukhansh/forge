#!/bin/bash
set -euo pipefail

mkdir -p reports

echo "========== UNIT + INTEGRATION TESTS =========="

docker pull alpine:3

FORGE_DOCKER_TESTS=1 go test \
    -count=1 \
    -coverpkg=./... \
    -coverprofile=reports/coverage.out \
    ./...

echo "========== COVERAGE GATE =========="

go tool cover -func=reports/coverage.out | tee reports/coverage.txt

COVERAGE=$(awk '/^total:/ {gsub(/%/, "", $NF); print $NF}' reports/coverage.txt)

awk -v actual="$COVERAGE" 'BEGIN {
    if (actual == "" || actual < 60) {
        print "COVERAGE_GATE_FAILED:", actual "%"
        exit 1
    }
    print "COVERAGE_GATE_PASSED:", actual "%"
}'

echo "========== STATIC ANALYSIS =========="

go vet ./...

echo "========== RACE DETECTION =========="

FORGE_DOCKER_TESTS=1 CGO_ENABLED=1 \
    go test -race -count=1 ./...

echo "QUALITY_GATES_PASSED"
