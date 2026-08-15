#!/usr/bin/env bash
# why: headless の agy は承認を得られないツールを soft-deny して exit 0 で素通りするため、
#      終了コードでは成否を判定できない。git diff を唯一の判定材料にし、許可パス外の書き換えは
#      その場で捨てる。CLI 差し替えに備えて呼び出しは DOCS_AGENT_CMD 越しにする。
set -uo pipefail

cd "$(dirname "$0")/../.." || exit 1

doc="${1:-}"
if [ -z "$doc" ] || [ ! -f "$doc" ]; then
  echo "usage: $0 <doc-path>" >&2
  exit 2
fi

agent_cmd="${DOCS_AGENT_CMD:-agy}"
read -ra agent_args <<<"${DOCS_AGENT_ARGS:---output-format json}"

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "作業ツリーに未コミットの変更があります。差分で判定できないため中断します。" >&2
  exit 2
fi

if ! command -v "$agent_cmd" >/dev/null 2>&1; then
  echo "$agent_cmd が見つかりません (DOCS_AGENT_CMD で差し替え可能)" >&2
  exit 2
fi

drift="$(bash scripts/docs/detect-drift.sh "$doc")"

prompt="あなたはこのリポジトリのドキュメント保守担当です。$doc を実装の現状に合わせて更新してください。

# 守ること
- 書き換えてよいのは $doc だけです。他のファイルは絶対に触らないでください。
- doc/adr/ の新規作成・Status 変更は禁止です。設計判断は人間が起こします。
- 実装ファイルを読んで裏が取れたことだけ書いてください。推測で書かないでください。
- 既存の文体 (敬体でない説明文・日本語) と見出し構成を保ってください。
- <!-- BEGIN generated: --> と <!-- END generated: --> に挟まれた範囲は生成物です。触らないでください。
- 変更が不要なら何も書き換えないでください。

# $doc の最終更新以降に変わった実装
$drift

# 進め方
上の changed_files のうち $doc の記述に関係するものを読み、記述と実装が食い違う箇所だけを直してください。"

log="$(mktemp)"
trap 'rm -f "$log"' EXIT

"$agent_cmd" -p "$prompt" "${agent_args[@]}" >"$log" 2>&1
agent_status=$?

changed="$(git diff --name-only)"

# 何も変わらなかったときこそ agy の出力が唯一の手がかりになる (soft-deny か、本当に変更不要か)。
if [ -z "$changed" ]; then
  echo "変更なし: $doc (agent exit=$agent_status)"
  cat "$log"
  exit 0
fi

if [ "$changed" != "$doc" ]; then
  echo "許可パス外が書き換えられたため破棄します: $changed" >&2
  cat "$log" >&2
  git checkout -- .
  exit 1
fi

echo "更新あり: $doc"
git --no-pager diff --stat -- "$doc"
