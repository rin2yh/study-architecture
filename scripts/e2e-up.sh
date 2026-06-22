#!/usr/bin/env bash
# Playwright の webServer から起動される。store / backoffice を listen させたままフォアグラウンドで
# 保持し、Playwright の url ポーリングを起動完了の検知に使う (teardown 時に Playwright が停止する)。
set -euo pipefail

docker compose up -d --wait db-customer db-ops
./scripts/migrate.sh

# 並列 build で docker daemon の I/O が詰まるのを避けるため 1 つずつ build する (.claude/rules/docker.md)。
for svc in product order payment member shipping shipping-worker; do
  docker compose build "$svc"
done
docker compose --profile external build store
docker compose --profile internal build backoffice

exec docker compose --profile external --profile internal up --no-build
