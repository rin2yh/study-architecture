#!/usr/bin/env bash
# why: 「全部見て直して」を AI に投げると差分が大きくなりレビュー不能になる。ドキュメント単位で
#      「最終更新以降に何が変わったか」だけを構造化して渡し、1 実行 1 ドキュメントに閉じるための入力を作る。
set -uo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root" || exit 1

impl_paths=(server client compose.yaml infra scripts mise.toml)
max_commits=40
max_files=60

docs=(
  "README.md"
  "client/README.md"
  "client/e2e/README.md"
  "client/app/ui/README.md"
  "doc/ops/runbook.md"
  "doc/ops/dashboards.md"
)

only="${1:-}"

emit_doc() {
  local doc="$1" since commits files count
  since="$(git log -1 --format=%cI -- "$doc")"
  [ -n "$since" ] || return 1

  commits="$(git log --since="$since" --format='%H%x09%s' -- "${impl_paths[@]}" |
    head -n "$max_commits" |
    jq -R -s 'split("\n") | map(select(length > 0)) | map(split("\t") | {sha: .[0][0:12], subject: .[1]})')"
  count="$(git log --since="$since" --oneline -- "${impl_paths[@]}" | wc -l | tr -d ' ')"
  files="$(git log --since="$since" --name-only --format= -- "${impl_paths[@]}" |
    grep -v '^$' | sort -u | head -n "$max_files" |
    jq -R -s 'split("\n") | map(select(length > 0))')"

  jq -n \
    --arg path "$doc" \
    --arg last_updated "$since" \
    --argjson commit_count "$count" \
    --argjson commits "$commits" \
    --argjson changed_files "$files" \
    '{path: $path, last_updated: $last_updated, commit_count: $commit_count, commits: $commits, changed_files: $changed_files}'
}

{
  for doc in "${docs[@]}"; do
    [ -f "$doc" ] || continue
    [ -n "$only" ] && [ "$doc" != "$only" ] && continue
    emit_doc "$doc"
  done
} | jq -s '{docs: (. | sort_by(-.commit_count))}'
