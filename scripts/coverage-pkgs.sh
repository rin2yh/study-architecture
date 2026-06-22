#!/usr/bin/env bash
# go test -coverpkg に渡す「カバレッジ計測対象」パッケージを導出する。
# 引数は go list に渡すパッケージスコープ (例: ./server/product/... ./server/internal/...)。
set -euo pipefail

# 除外対象を path 正規表現で列挙すると新サービス・新生成物のたびに腐るので、
# パッケージの性質から導出する: main は起動点、Code generated は生成物、test ヘルパは
# テスト専用。残り (handler/rdb/gateway/auth/stub 等) が計測対象。
go list -f '{{if ne .Name "main"}}{{.ImportPath}}{{"\t"}}{{.Dir}}{{end}}' "$@" |
  while IFS=$'\t' read -r import_path dir; do
    [ -z "$import_path" ] && continue
    case "$import_path" in */internal/test/*) continue ;; esac
    grep -lqE '^// Code generated .* DO NOT EDIT' "$dir"/*.go 2>/dev/null && continue
    echo "$import_path"
  done | paste -sd, -
