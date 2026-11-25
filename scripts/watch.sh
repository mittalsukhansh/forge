#!/bin/bash
set -euo pipefail

mkdir -p reports

STATE="reports/last-release"
LAST_ATTEMPT=""

if [ ! -f "$STATE" ]; then
    git rev-parse HEAD > "$STATE"
fi

echo "FORGE RELEASE WATCHER STARTED"
echo "Last released commit: $(cat "$STATE")"

while true; do
    git fetch origin main

    NEW_COMMIT=$(git rev-parse origin/main)
    LAST_RELEASE=$(cat "$STATE")

    if [ "$NEW_COMMIT" = "$LAST_RELEASE" ]; then

        echo "$(date -u) NO_CHANGES"

    elif [ "$NEW_COMMIT" = "$LAST_ATTEMPT" ]; then

        echo "$(date -u) PREVIOUS_BUILD_FAILED"

    else
        echo "NEW_COMMIT: $NEW_COMMIT"

        git merge --ff-only origin/main

        LAST_ATTEMPT="$NEW_COMMIT"
        LOG="reports/release-${NEW_COMMIT:0:12}.log"

        if python3 scripts/runner.py 2>&1 | tee "$LOG"; then
            echo "$NEW_COMMIT" > "$STATE"

            echo "AUTOMATED_RELEASE_SUCCESS: $NEW_COMMIT"
        else
            echo "AUTOMATED_RELEASE_FAILED: $NEW_COMMIT"
        fi
    fi

    sleep 30
done
