#!/usr/bin/env bash
# why: ADR 参照と一覧の整合は .claude/rules/adr.md の「作成後にやること」を人手で守る前提だったが、
#      漏れると壊れリンクとして残り続ける。機械で弾いて ADR-[[202607020343]] の枠に載せる。
set -uo pipefail

cd "$(dirname "$0")/../.." || exit 1

adr_dir="doc/adr"
index="$adr_dir/README.md"
violations=0

report() {
  printf '%s\n' "$1" >&2
  violations=$((violations + 1))
}

declare -A adr_path_by_id adr_status_by_id

# 参照は 400 件超あり、引くたびに find/grep を起動すると 1 実行あたり数百プロセスになる。
load_adrs() {
  local path name line status first
  for path in "$adr_dir"/[0-9]*.md; do
    name="${path##*/}"
    adr_path_by_id["${name:0:12}"]="$path"
  done

  while IFS= read -r line; do
    path="${line%%:*}"
    status="${line#*- Status:}"
    read -r first _ <<<"$status"
    name="${path##*/}"
    adr_status_by_id["${name:0:12}"]="$first"
  done < <(grep -m1 -H -E '^- Status:' "$adr_dir"/[0-9]*.md)
}

# 参照は Markdown だけでなくコード中コメントにも張る規約のため、追跡対象の全ファイルを見る。
check_adr_refs() {
  local file id
  while IFS=: read -r file id; do
    [ -n "${adr_path_by_id[$id]:-}" ] && continue
    report "壊れた ADR 参照: $file が ADR-[[$id]] を参照していますが $adr_dir に実在しません"
  done < <(
    git grep -oI -E 'ADR-\[\[[0-9]{12}\]\]' -- . |
      sed -E 's/^(.+):ADR-\[\[([0-9]{12})\]\]$/\1:\2/' | sort -u
  )
}

check_index_coverage() {
  local index_body path name linked
  index_body="$(<"$index")"

  for path in "$adr_dir"/[0-9]*.md; do
    name="${path##*/}"
    [[ $index_body == *"($name)"* ]] && continue
    report "一覧漏れ: $path が $index の表にありません ($(head -n1 "$path" | sed -E 's/^# //'))"
  done

  while read -r linked; do
    [ -f "$adr_dir/$linked" ] && continue
    report "一覧の壊れ行: $index が実在しない $linked を指しています"
  done < <(grep -oE '\(([0-9]{12}-[a-z0-9-]+\.md)\)' "$index" | tr -d '()' | sort -u)
}

# Status は表と本文の二重管理になるため、先頭語 (Accepted / Superseded / Proposed) だけ突き合わせる。
check_status_consistency() {
  local id_cell status_cell id index_status body_status
  while IFS='|' read -r _ id_cell status_cell _; do
    [[ $id_cell =~ ([0-9]{12}) ]] || continue
    id="${BASH_REMATCH[1]}"
    body_status="${adr_status_by_id[$id]:-}"
    [ -n "$body_status" ] || continue
    read -r index_status _ <<<"$status_cell"
    [ "$index_status" = "$body_status" ] && continue
    report "Status 不一致: ADR-$id は一覧が '$index_status'、本文が '$body_status'"
  done < <(grep -E '^\| \[[0-9]{12}\]' "$index")
}

check_relative_links() {
  local file link target dir
  while IFS=: read -r file link; do
    target="${link%%#*}"
    # ${file%/*} はスラッシュを含まないリポジトリ直下のパスを縮められない。
    dir="${file%/*}"
    [ "$dir" = "$file" ] && dir="."
    [ -e "$dir/$target" ] && continue
    report "壊れた相対リンク: $file の ($link) が存在しません"
  done < <(
    git grep -oI -E '\]\([^)]+\)' -- '*.md' |
      sed -E 's/^(.+):\]\((.+)\)$/\1:\2/' |
      grep -vE ':(https?:|mailto:|#)' | sort -u
  )
}

load_adrs
check_adr_refs
check_index_coverage
check_status_consistency
check_relative_links

if [ "$violations" -gt 0 ]; then
  printf '\nドキュメント参照の違反 %d 件。.claude/rules/adr.md の「作成後にやること」を確認してください。\n' "$violations" >&2
  exit 1
fi

echo "docs refs: OK"
