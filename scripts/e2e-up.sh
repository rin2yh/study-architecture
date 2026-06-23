#!/usr/bin/env bash
# Playwright の setup project から呼ばれ、指定フロント (store / backoffice) のスタックを detached で
# 起動する。停止は teardown project の e2e-down.sh が行う。
set -euo pipefail

target="${1:?usage: e2e-up.sh <store|backoffice>}"
case "$target" in
  store) profile=external ;;
  backoffice) profile=internal ;;
  *)
    echo "unknown target: $target (want store|backoffice)" >&2
    exit 1
    ;;
esac

docker compose up -d --wait db-customer db-ops
./scripts/migrate.sh

# 並列 build で docker daemon の I/O が詰まるのを避けるため 1 つずつ build する (.claude/rules/docker.md)。
for svc in product order payment member shipping shipping-worker; do
  docker compose build "$svc"
done
docker compose --profile "$profile" build "$target"

docker compose --profile "$profile" up -d --wait
