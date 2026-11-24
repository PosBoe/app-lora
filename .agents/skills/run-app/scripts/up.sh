#!/usr/bin/env bash
# up.sh starts the app on a scratch PostgreSQL database: the Go server and the
# Vite dev server. The last line it prints is the state file for down.sh.
set -euo pipefail

root=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
db=lora_scratch_$(date +%H%M%S)
work=${TMPDIR:-/tmp}/run-app-$db
http_port=${RUN_APP_HTTP_PORT:-18080}
vite_port=${RUN_APP_VITE_PORT:-15173}

for port in "$http_port" "$vite_port"; do
  if ss -ltn "sport = :$port" | grep -q LISTEN; then
    echo "up.sh: port $port has a listener; see ss -ltnp \"sport = :$port\"" >&2
    exit 1
  fi
done

mkdir -p "$work"

src=$root/config/dev.yml
[ -f "$src" ] || src=$root/config/dev.default.yml
sed -e "s#^\(  dsn: .*/\)lora_dev#\1$db#" -e "s#^  http: .*#  http: :$http_port#" "$src" >"$work/config.yml"
if ! grep -q "/$db" "$work/config.yml"; then
  echo "up.sh: cannot point pg.dsn in $src at $db" >&2
  exit 1
fi

cat >"$work/state.env" <<STATE
DB=$db
WORK=$work
HTTP_PORT=$http_port
VITE_PORT=$vite_port
STATE

echo "building ./cmd/lora"
(cd "$root" && go build -tags=invariants -o "$work/lora" ./cmd/lora)

export LORA_CONFIG=$work/config.yml
"$work/lora" pg create_db >>"$work/setup.log" 2>&1
"$work/lora" migrate up >>"$work/setup.log" 2>&1

# setsid gives each server its own process group, so down.sh stops the whole
# tree (pnpm starts vite as a child).
setsid "$work/lora" serve >"$work/serve.log" 2>&1 &
echo "SERVE_PGID=$!" >>"$work/state.env"

(
  cd "$root/vue"
  [ -d node_modules ] || pnpm install >>"$work/setup.log" 2>&1
  VITE_DEV_PROXY_API=http://localhost:$http_port \
    setsid pnpm exec vite --port "$vite_port" --strictPort >"$work/vite.log" 2>&1 &
  echo "VITE_PGID=$!" >>"$work/state.env"
)

wait_ok() {
  local url=$1
  for _ in $(seq 1 120); do
    if [ "$(curl -s -o /dev/null -w '%{http_code}' "$url")" = 200 ]; then
      return 0
    fi
    sleep 0.5
  done
  echo "up.sh: $url did not answer 200; see the logs in $work" >&2
  return 1
}

wait_ok "http://localhost:$http_port/api/v1/health"
wait_ok "http://localhost:$vite_port/"

echo "app: http://localhost:$vite_port (api :$http_port, database $db)"
echo "$work/state.env"
