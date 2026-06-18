#!/usr/bin/env bash
# host から goose で migration を流す。引数 1 つで個別サービス、無しで 5 サービス全部。
# DATABASE_URL_CUSTOMER / DATABASE_URL_OPS で DSN を上書きできる (CI などで使う)。
set -euo pipefail

: "${DATABASE_URL_CUSTOMER:=postgres://ec:ec_pass@localhost:5432/ec_customer?sslmode=disable}"
: "${DATABASE_URL_OPS:=postgres://ec:ec_pass@localhost:5433/ec_ops?sslmode=disable}"

migrate_one() {
  local svc="$1"
  local dsn
  case "$svc" in
    order|payment|member) dsn="$DATABASE_URL_CUSTOMER" ;;
    product|shipping)     dsn="$DATABASE_URL_OPS" ;;
    *) echo "unknown service: $svc" >&2; return 1 ;;
  esac
  go tool goose -table "goose_${svc}_version" -dir "server/${svc}/db/migration" postgres "$dsn" up
}

if [ "$#" -ge 1 ]; then
  migrate_one "$1"
else
  for svc in order payment member product shipping; do
    migrate_one "$svc"
  done
fi
