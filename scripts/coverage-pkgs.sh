#!/usr/bin/env bash
# go test -coverpkg に渡す「カバレッジ計測対象」パッケージを導出する。
set -euo pipefail

# 除外対象を path 正規表現で列挙すると新サービス・新生成物のたびに腐るので、性質から導出する。
go list -f '{{if ne .Name "main"}}{{.ImportPath}}{{"\t"}}{{.Dir}}{{end}}' "$@" |
  while IFS=$'\t' read -r import_path dir; do
    [ -z "$import_path" ] && continue
    case "$import_path" in */internal/test/*) continue ;; esac
    grep -lqE '^// Code generated .* DO NOT EDIT' "$dir"/*.go 2>/dev/null && continue
    echo "$import_path"
  done | paste -sd, -
