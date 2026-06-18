#!/usr/bin/env bash
# host から goose で migration を流す。引数 1 つで個別サービス、無しで 5 サービス全部。
# DATABASE_URL_CUSTOMER / DATABASE_URL_OPS で DSN を上書きできる (CI などで使う)。
set -euo pipefail

: "${DATABASE_URL_CUSTOMER:=postgres://ec:ec_pass@localhost:5432/ec_customer?sslmode=disable}"
: "${DATABASE_URL_OPS:=postgres://ec:ec_pass@localhost:5433/ec_ops?sslmode=disable}"

# 5 サービスを go tool goose で並列に呼ぶと、各プロセスが goose バイナリのビルド/リンクを
# 同時に走らせて激しく競合する (>5min)。先に一度だけビルドし、そのバイナリを使い回す。
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
# ビルド済みバイナリなら適用は DB 往復だけなので並列で短縮できる。
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
