#!/usr/bin/env bash
# Playwright の webServer から起動される。引数のフロントエンド (store / backoffice) を listen
# させたままフォアグラウンドで保持し、Playwright の url ポーリングを起動完了の検知に使う
# (teardown 時に Playwright が停止する)。
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

compose=(docker compose)
if [ -n "${CI:-}" ]; then
  # CI では docker layer を GitHub Actions cache に出し入れして build を短縮する (compose.ci.yaml)。
  compose=(docker compose -f compose.yaml -f compose.ci.yaml)
  export COMPOSE_BAKE=true
fi

"${compose[@]}" up -d --wait db-customer db-ops
./scripts/migrate.sh

# 並列 build で docker daemon の I/O が詰まるのを避けるため 1 つずつ build する (.claude/rules/docker.md)。
for svc in product order payment member shipping shipping-worker; do
  "${compose[@]}" build "$svc"
done
"${compose[@]}" --profile "$profile" build "$target"

exec "${compose[@]}" --profile "$profile" up --no-build
