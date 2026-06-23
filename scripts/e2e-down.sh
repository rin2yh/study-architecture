#!/usr/bin/env bash
# Playwright の teardown project から呼ばれる。profile 配下のフロントも落とすため両 profile を指定する。
set -euo pipefail

docker compose --profile external --profile internal down
