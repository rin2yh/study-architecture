#!/usr/bin/env bash
# E2E 用スタックを起動する。
set -euo pipefail

cd "$(dirname "$0")/.."

app="${1:?usage: e2e-up.sh <store|backoffice>}"
case "$app" in
  store) profile=external ;;
  backoffice) profile=internal ;;
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
