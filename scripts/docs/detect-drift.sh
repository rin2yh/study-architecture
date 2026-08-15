#!/usr/bin/env bash
# why: 「全部見て直して」を AI に投げると差分が大きくなりレビュー不能になる。ドキュメント単位で
#      「最終更新以降に何が変わったか」だけを構造化して渡し、1 実行 1 ドキュメントに閉じる。
set -uo pipefail

cd "$(dirname "$0")/../.." || exit 1

doc="${1:?usage: detect-drift.sh <doc-path>}"
if [ ! -f "$doc" ]; then
  echo "ドキュメントが見つかりません: $doc" >&2
  exit 2
fi

max_commits=40
max_files=60

since="$(git log -1 --format=%cI -- "$doc")"
if [ -z "$since" ]; then
  echo "$doc の履歴がありません" >&2
  exit 2
fi

# 実装側を許可リストで数えると、新しいトップレベルが増えたときに黙って漏れる。
log="$(git log --since="$since" --format='%x01%H%x09%s' --name-only -- . ':(exclude)*.md')"

jq -Rn \
  --arg path "$doc" \
  --arg last_updated "$since" \
  --argjson max_commits "$max_commits" \
  --argjson max_files "$max_files" '
  [inputs] as $lines
  | ($lines
     | map(select(startswith("\u0001"))
           | ltrimstr("\u0001") | split("\t")
           | {sha: .[0][0:12], subject: .[1]})) as $commits
  | ($lines
     | map(select(startswith("\u0001") | not) | select(length > 0))
     | unique) as $files
  | {
      path: $path,
      last_updated: $last_updated,
      commit_count: ($commits | length),
      commits: $commits[0:$max_commits],
      changed_files: $files[0:$max_files]
    }
' <<<"$log"
