#!/bin/bash
set -euo pipefail

IMAGE="${1:?Image name required}"
NAME="forge-smoke-$$"

cleanup() {
    docker rm -fv "$NAME" >/dev/null 2>&1 || true
}

trap cleanup EXIT

echo "========== CONTAINER SMOKE TEST =========="

docker run -d \
    --name "$NAME" \
    --group-add "$(stat -c '%g' /var/run/docker.sock)" \
    -p 127.0.0.1::8080 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    "$IMAGE"

PORT=$(docker port "$NAME" 8080/tcp | awk -F: '{print $NF}')
BASE="http://127.0.0.1:$PORT"

READY=0

for i in $(seq 1 30); do
    if curl -fsS "$BASE/health"; then
        READY=1
        break
    fi

    sleep 1
done

test "$READY" = "1"

RESPONSE=$(python3 -c \
    'import json; print(json.dumps({"pipeline":open("examples/pipeline.yml").read()}))' \
    | curl -fsS -X POST "$BASE/api/jobs" \
        -H "Content-Type: application/json" \
        --data-binary @-)

ID=$(echo "$RESPONSE" | python3 -c \
    'import json,sys; print(json.load(sys.stdin)["id"])')

for i in $(seq 1 60); do
    RESULT=$(curl -fsS "$BASE/api/jobs/$ID")

    STATUS=$(echo "$RESULT" | python3 -c \
        'import json,sys; print(json.load(sys.stdin)["status"])')

    echo "SMOKE_JOB_STATUS: $STATUS"

    if [ "$STATUS" = "succeeded" ]; then
        echo "SMOKE_TEST_PASSED"
        exit 0
    fi

    if [ "$STATUS" = "failed" ]; then
        echo "$RESULT"
        exit 1
    fi

    sleep 2
done

echo "SMOKE_TEST_TIMEOUT"
exit 1
