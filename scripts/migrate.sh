#!/usr/bin/env bash
set -euo pipefail

: "${DATABASE_URL_ORDER:=postgres://ec:ec_pass@localhost:5432/ec_order?sslmode=disable}"
: "${DATABASE_URL_PAYMENT:=postgres://ec:ec_pass@localhost:5434/ec_payment?sslmode=disable}"
: "${DATABASE_URL_MEMBER:=postgres://ec:ec_pass@localhost:5436/ec_member?sslmode=disable}"
: "${DATABASE_URL_SHIPPING:=postgres://ec:ec_pass@localhost:5435/ec_shipping?sslmode=disable}"
: "${DATABASE_URL_PRODUCT:=postgres://ec:ec_pass@localhost:5433/ec_product?sslmode=disable}"

# go tool goose は全 DB ドライバを毎回ビルドして遅いため、prebuilt があればそれを使う。
goose=(goose)
command -v goose >/dev/null 2>&1 || goose=(go tool goose)

migrate_one() {
  local svc="$1"
  local dsn
  case "$svc" in
    order)    dsn="$DATABASE_URL_ORDER" ;;
    payment)  dsn="$DATABASE_URL_PAYMENT" ;;
    member)   dsn="$DATABASE_URL_MEMBER" ;;
    shipping) dsn="$DATABASE_URL_SHIPPING" ;;
    product)  dsn="$DATABASE_URL_PRODUCT" ;;
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
