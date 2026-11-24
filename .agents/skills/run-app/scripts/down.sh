#!/usr/bin/env bash
# down.sh stops a run that up.sh started and drops its scratch database.
#
#   down.sh <state file> [--keep-logs]
set -euo pipefail

state=${1:?usage: down.sh <state file> [--keep-logs]}
keep=${2:-}
# shellcheck disable=SC1090
. "$state"

for pgid in "${VITE_PGID:-}" "${SERVE_PGID:-}"; do
  [ -n "$pgid" ] || continue
  kill -TERM -- "-$pgid" 2>/dev/null || true
done
sleep 1
for pgid in "${VITE_PGID:-}" "${SERVE_PGID:-}"; do
  [ -n "$pgid" ] || continue
  kill -KILL -- "-$pgid" 2>/dev/null || true
done

LORA_CONFIG=$WORK/config.yml "$WORK/lora" pg drop_db

if [ "$keep" = "--keep-logs" ]; then
  rm -f "$WORK/lora"
  echo "kept $WORK"
else
  rm -rf "$WORK"
fi
