#!/usr/bin/env bash
# why: ADR 参照と一覧の整合は .claude/rules/adr.md の「作成後にやること」を人手で守る前提だったが、
#      漏れると壊れリンクとして残り続ける。機械で弾いて ADR-[[202607020343]] の枠に載せる。
set -uo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root" || exit 1

adr_dir="doc/adr"
index="$adr_dir/README.md"
violations=0

report() {
  printf '%s\n' "$1" >&2
  violations=$((violations + 1))
}

adr_path_for() {
  local id="$1"
  find "$adr_dir" -maxdepth 1 -name "$id-*.md" -print -quit
}

# 参照は Markdown だけでなくコード中コメントにも張る規約のため、追跡対象の全ファイルを見る。
check_adr_refs() {
  local file id path
  while IFS=: read -r file id; do
    path="$(adr_path_for "$id")"
    [ -n "$path" ] && continue
    report "壊れた ADR 参照: $file が ADR-[[$id]] を参照していますが $adr_dir に実在しません"
  done < <(
    git grep -oI -E 'ADR-\[\[[0-9]{12}\]\]' -- . ':(exclude)scripts/docs' |
      sed -E 's/^(.+):ADR-\[\[([0-9]{12})\]\]$/\1:\2/' | sort -u
  )
}

check_index_coverage() {
  local id title
  for path in "$adr_dir"/[0-9]*.md; do
    id="$(basename "$path" | cut -c1-12)"
    grep -q "($(basename "$path"))" "$index" && continue
    title="$(head -n1 "$path" | sed -E 's/^# //')"
    report "一覧漏れ: $path が $index の表にありません ($title)"
  done

  while read -r linked; do
    [ -f "$adr_dir/$linked" ] && continue
    report "一覧の壊れ行: $index が実在しない $linked を指しています"
  done < <(grep -oE '\(([0-9]{12}-[a-z0-9-]+\.md)\)' "$index" | tr -d '()' | sort -u)
}

# Status は表と本文の二重管理になるため、先頭語 (Accepted / Superseded / Proposed) だけ突き合わせる。
check_status_consistency() {
  local file id body_status index_status
  while IFS='|' read -r _ id_cell status_cell _; do
    id="$(printf '%s' "$id_cell" | grep -oE '[0-9]{12}' | head -n1)"
    [ -n "$id" ] || continue
    file="$(adr_path_for "$id")"
    [ -n "$file" ] || continue
    index_status="$(printf '%s' "$status_cell" | awk '{print $1}')"
    body_status="$(grep -m1 -E '^- Status:' "$file" | sed -E 's/^- Status:[[:space:]]*//' | awk '{print $1}')"
    [ "$index_status" = "$body_status" ] && continue
    report "Status 不一致: ADR-$id は一覧が '$index_status'、本文が '$body_status'"
  done < <(grep -E '^\| \[[0-9]{12}\]' "$index")
}

check_relative_links() {
  local file link target dir
  while IFS=: read -r file link; do
    dir="$(dirname "$file")"
    target="${link%%#*}"
    [ -n "$target" ] || continue
    [ -e "$dir/$target" ] && continue
    report "壊れた相対リンク: $file の ($link) が存在しません"
  done < <(
    git grep -oI -E '\]\([^)]+\)' -- '*.md' |
      sed -E 's/^(.+):\]\((.+)\)$/\1:\2/' |
      grep -vE ':(https?:|mailto:|#)' | sort -u
  )
}

check_adr_refs
check_index_coverage
check_status_consistency
check_relative_links

if [ "$violations" -gt 0 ]; then
  printf '\nドキュメント参照の違反 %d 件。.claude/rules/adr.md の「作成後にやること」を確認してください。\n' "$violations" >&2
  exit 1
fi

echo "docs refs: OK"
