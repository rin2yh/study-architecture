#!/usr/bin/env bash
# 起動だけでなく migrate / grant / seed 前提まで揃えてから up する (E2E は実 DB と実ロールを要する)。
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

docker compose up -d --wait db-order db-payment db-member db-shipping db-product db-inventory
./scripts/migrate.sh
./scripts/grant.sh

# 並列 build で docker daemon の I/O が詰まるのを避ける (.claude/rules/docker.md)。
for svc in product order payment member shipping shipping-worker inventory inventory-worker; do
  docker compose build "$svc"
done
docker compose --profile "$profile" build "$app"
docker compose --profile "$profile" up -d --wait
