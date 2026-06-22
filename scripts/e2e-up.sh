#!/usr/bin/env bash
# Playwright の webServer から起動される。store を listen させたままフォアグラウンドで保持し、
# Playwright の url ポーリングを起動完了の検知に使う (teardown 時に Playwright が停止する)。
set -euo pipefail

docker compose up -d --wait db-customer db-ops
./scripts/migrate.sh
exec docker compose --profile external up --build
