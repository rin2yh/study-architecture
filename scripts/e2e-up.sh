#!/usr/bin/env bash
# E2E 用スタックを起動する。Playwright を呼ぶ前に mise タスク / CI から実行する。
set -euo pipefail

cd "$(dirname "$0")/.."

app="${1:?usage: e2e-up.sh <store|backoffice>}"
case "$app" in
  store) profile=external; port=5173 ;;
  backoffice) profile=internal; port=5175 ;;
  *)
    echo "unknown app: $app (want store|backoffice)" >&2
    exit 1
    ;;
esac

docker compose up -d --wait db-customer db-ops
./scripts/migrate.sh

# 並列 build で docker daemon の I/O が詰まるのを避けるため 1 つずつ build する (.claude/rules/docker.md)。
for svc in product order payment member shipping shipping-worker; do
  docker compose build "$svc"
done
docker compose --profile "$profile" build "$app"
docker compose --profile "$profile" up -d --wait

# frontend は healthcheck を持たない (compose の --wait では待てない)。
deadline=$((SECONDS + 120))
until curl -fsS -o /dev/null "http://localhost:${port}/"; do
  if [ "$SECONDS" -ge "$deadline" ]; then
    echo "timeout waiting for frontend on :${port}" >&2
    exit 1
  fi
  sleep 1
done
