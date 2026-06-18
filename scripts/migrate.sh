#!/usr/bin/env bash
# host から goose で migration を流す。引数 1 つで個別サービス、無しで 5 サービス全部。
# DATABASE_URL_CUSTOMER / DATABASE_URL_OPS で DSN を上書きできる (CI などで使う)。
set -euo pipefail

: "${DATABASE_URL_CUSTOMER:=postgres://ec:ec_pass@localhost:5432/ec_customer?sslmode=disable}"
: "${DATABASE_URL_OPS:=postgres://ec:ec_pass@localhost:5433/ec_ops?sslmode=disable}"

# goose を 5 回呼ぶので、go tool 経由だと毎回リンクが走る。一度だけビルドして使い回す。
goose_bin="$(mktemp -d)/goose"
go build -o "$goose_bin" github.com/pressly/goose/v3/cmd/goose

migrate_one() {
  local svc="$1"
  local dsn
  case "$svc" in
    order|payment|member) dsn="$DATABASE_URL_CUSTOMER" ;;
    product|shipping)     dsn="$DATABASE_URL_OPS" ;;
    *) echo "unknown service: $svc" >&2; return 1 ;;
  esac
  "$goose_bin" -table "goose_${svc}_version" -dir "server/${svc}/db/migration" postgres "$dsn" up
}

if [ "$#" -ge 1 ]; then
  migrate_one "$1"
  exit 0
fi

# 各サービスは別スキーマ + 別 goose version table を使うので、同一 DB へ同時に流しても衝突しない。
# 直列だと DB 往復の待ちが 5 回分積み上がるため並列化して CI 時間を縮める。
pids=()
for svc in order payment member product shipping; do
  migrate_one "$svc" &
  pids+=("$!")
done

status=0
for pid in "${pids[@]}"; do
  wait "$pid" || status=1
done
exit "$status"
