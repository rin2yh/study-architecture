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
./scripts/grant.sh

# 逐次 build は OrbStack (ローカル) の daemon I/O 競合対策 (.claude/rules/docker.md)。CI runner の
# dockerd には当てはまらず、逐次だと UI/backend のビルドが直列化して支配項になるため、CI では
# 1 ショット並列 build に切り替える (buildkit が共通 go.mod レイヤを 1 回に dedup する)。
if [ -n "${CI:-}" ]; then
  docker compose --profile "$profile" build
else
  for svc in product order payment member shipping shipping-worker; do
    docker compose build "$svc"
  done
  docker compose --profile "$profile" build "$app"
fi
docker compose --profile "$profile" up -d --wait
