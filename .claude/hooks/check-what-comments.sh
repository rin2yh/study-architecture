#!/usr/bin/env bash
# why: what/why の区別は意味判定で、静的解析では落とせない。判定は claude -p に委譲し、
#      .claude/rules/comments.md を唯一の規約ソースとして読み込む (規約をここへ転記しない)。
# PostToolUse(Edit|Write|MultiEdit) で呼ばれ、what コメントを見つけたら exit 2 で差し戻す。
set -uo pipefail

# 自分が起動する claude -p からの再入を断つ (settings 側でも hooks を無効化している)。
[ -n "${CLAUDE_WHAT_COMMENT_HOOK:-}" ] && exit 0

repo_root="${CLAUDE_PROJECT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
rules="$repo_root/.claude/rules/comments.md"
[ -f "$rules" ] || exit 0

payload="$(cat)"

file="$(printf '%s' "$payload" | jq -r '.tool_input.file_path // empty')"
[ -n "$file" ] || exit 0

case "$file" in
  *.go|*.ts|*.tsx|*.js|*.jsx|*.sql|*.yaml|*.yml|*.sh) ;;
  *) exit 0 ;;
esac

# 生成コードは comments.md の例外。
case "$file" in
  *.gen.go|*.sql.go) exit 0 ;;
esac
[ -f "$file" ] && head -n 5 "$file" | grep -q 'Code generated' && exit 0

added="$(printf '%s' "$payload" | jq -r '
  .tool_input.content
  // .tool_input.new_string
  // ([.tool_input.edits[]?.new_string] | join("\n"))
  // empty')"
[ -n "$added" ] || exit 0

# コメントらしき行が無い編集は LLM を呼ばずに抜ける (呼び出しの大半をここで節約)。
case "$file" in
  *.yaml|*.yml|*.sh) marker='#' ;;
  *.sql)             marker='--' ;;
  *)                 marker='//|/\*' ;;
esac
# grep に "--" を素で渡すとオプション終端と解釈されるため -e で必ずパターン扱いにする。
printf '%s' "$added" | grep -qE -e "$marker" || exit 0

snippet="$(printf '%s' "$added" | head -n 400)"
prompt="あなたはコメント規約の linter です。下記「規約」に従い、「対象コード」のコメントのうち
what コメント (コードを読めば分かる処理の言い換え) だけを指摘してください。
why コメント・doc コメント・生成マーカー・ADR 参照 ([[NNNN]]) は指摘しないこと。
とくに Go の公開識別子に付く doc コメント (識別子名で始まる行) は規約の明示的な例外なので、
署名から読み取れる内容を含んでいても指摘しない。

# 規約
$(cat "$rules")

# 対象コード ($file)
$snippet

# 出力 (厳守)
what コメントが無ければ最初の行に OK とだけ出力する。
あれば各違反を 1 行ずつ \"対象コメント => 是正方針\" で列挙し、前置き・後置きは書かない。"

result="$(printf '%s' "$prompt" | CLAUDE_WHAT_COMMENT_HOOK=1 timeout 90 \
  claude -p --model claude-haiku-4-5-20251001 \
  --settings '{"disableAllHooks":true}' 2>/dev/null)" || exit 0

# 判定不能・空応答は fail-open (正当な編集を止めない)。
printf '%s' "$result" | head -n1 | grep -qE '^[[:space:]]*OK\b' && exit 0
[ -z "${result//[[:space:]]/}" ] && exit 0

{
  echo "comments.md 違反 (what コメント) の可能性があります。why だけ残すよう修正してください:"
  printf '%s\n' "$result"
} >&2
exit 2
