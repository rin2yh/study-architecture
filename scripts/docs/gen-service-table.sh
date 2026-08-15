#!/usr/bin/env bash
# why: README のサービス表は手書きだったため実装から取り残された (inventory 欠落・コンテナ内ポートが
#      実体と違う)。compose.yaml を単一情報源にして腐る余地を消す (ADR-[[202606170901]] の延長)。
set -uo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root" || exit 1

readme="README.md"
begin_marker="<!-- BEGIN generated: services (scripts/docs/gen-service-table.sh) -->"
end_marker="<!-- END generated: services -->"

mode="${1:---check}"

# profiles 指定の無いサービスだけを既定で出すと store/backoffice/可観測性が落ちるため全 profile を有効化する。
compose_json() {
  docker compose --profile external --profile internal --profile observability config --format json 2>/dev/null
}

render_table() {
  local json="$1"
  {
    echo "$begin_marker"
    echo
    echo "| 区分 | 名前 | ホストポート | コンテナ内 | profile | ネットワーク |"
    echo "| --- | --- | --- | --- | --- | --- |"
    jq -r '
      def kind(name; profiles):
        if (name | startswith("db-")) then "1|データ"
        elif (name | endswith("-worker")) then "3|ワーカー"
        elif (name == "store" or name == "backoffice") then "4|UI"
        elif (profiles | index("observability")) then "5|可観測性"
        elif (name == "edge-proxy" or name == "broker") then "0|基盤"
        else "2|サービス" end;
      .services
      | to_entries
      | map(
          (.value.profiles // []) as $p
          | (kind(.key; $p) | split("|")) as $k
          | {
              sort: ($k[0] + "-" + .key),
              kind: $k[1],
              name: .key,
              host: ([(.value.ports // [])[] | (.published | tostring)] | join(" ")),
              target: ([(.value.ports // [])[] | (.target | tostring)] | join(" ")),
              profile: ($p | join(",")),
              nets: ((.value.networks // {}) | keys | join(", "))
            }
        )
      | sort_by(.sort)[]
      | "| \(.kind) | `\(.name)` | \(if .host == "" then "-" else .host end) | \(if .target == "" then "-" else .target end) | \(if .profile == "" then "既定" else .profile end) | \(.nets) |"
    ' <<<"$json"
    echo
    echo "$end_marker"
  }
}

json="$(compose_json)"
if [ -z "$json" ]; then
  echo "compose.yaml を解釈できませんでした (docker compose config が失敗)" >&2
  exit 1
fi

if ! grep -qF "$begin_marker" "$readme" || ! grep -qF "$end_marker" "$readme"; then
  echo "$readme に生成マーカーがありません: $begin_marker / $end_marker" >&2
  exit 1
fi

table="$(render_table "$json")"
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

awk -v begin="$begin_marker" -v end="$end_marker" -v table="$table" '
  index($0, begin) == 1 { print table; skip = 1; next }
  index($0, end) == 1 { skip = 0; next }
  !skip { print }
' "$readme" >"$tmp"

case "$mode" in
  --write)
    cp "$tmp" "$readme"
    echo "service table: $readme を更新しました"
    ;;
  --check)
    if diff -u "$readme" "$tmp" >/dev/null; then
      echo "service table: OK"
    else
      echo "README のサービス表が compose.yaml と乖離しています。scripts/docs/gen-service-table.sh --write を実行してください:" >&2
      diff -u "$readme" "$tmp" >&2
      exit 1
    fi
    ;;
  *)
    echo "usage: $0 [--check|--write]" >&2
    exit 2
    ;;
esac
