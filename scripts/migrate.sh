#!/usr/bin/env bash
set -euo pipefail

: "${DATABASE_URL_CUSTOMER:=postgres://ec:ec_pass@localhost:5432/ec_customer?sslmode=disable}"
: "${DATABASE_URL_PAYMENT:=postgres://ec:ec_pass@localhost:5434/ec_payment?sslmode=disable}"
: "${DATABASE_URL_OPS:=postgres://ec:ec_pass@localhost:5433/ec_ops?sslmode=disable}"

# go tool goose は全 DB ドライバを毎回ビルドして遅いため、prebuilt があればそれを使う。
goose=(goose)
command -v goose >/dev/null 2>&1 || goose=(go tool goose)

migrate_one() {
  local svc="$1"
  local dsn
  case "$svc" in
    order|member)     dsn="$DATABASE_URL_CUSTOMER" ;;
    payment)          dsn="$DATABASE_URL_PAYMENT" ;;
    product|shipping) dsn="$DATABASE_URL_OPS" ;;
    *) echo "unknown service: $svc" >&2; return 1 ;;
  esac
  "${goose[@]}" -table "goose_${svc}_version" -dir "server/${svc}/db/migration" postgres "$dsn" up
}

if [ "$#" -ge 1 ]; then
  migrate_one "$1"
else
  for svc in order payment member product shipping; do
    migrate_one "$svc"
  done
fi
