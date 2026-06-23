#!/usr/bin/env bash
# Playwright の teardown project から呼ばれる。profile 配下のフロントも落とすため。
set -euo pipefail

profile="${1:?usage: e2e-down.sh <profile>}"

docker compose --profile "$profile" down
