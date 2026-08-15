#!/usr/bin/env bash
# why: README のサービス表は手書きだったため実装から取り残された (inventory 欠落・コンテナ内ポートが
#      実体と違う)。compose.yaml を単一情報源にして腐る余地を消す (ADR-[[202606170901]] の延長)。
set -uo pipefail

cd "$(dirname "$0")/../.." || exit 1

readme="README.md"
begin_marker="<!-- BEGIN generated: services (scripts/docs/gen-service-table.sh) -->"
end_marker="<!-- END generated: services -->"

mode="${1:---check}"
case "$mode" in
  --check | --write) ;;
  *)
    echo "usage: $0 [--check|--write]" >&2
    exit 2
    ;;
esac

# profile 付きサービスは既定の config に出ないため、compose 自身に列挙させて全部有効化する
# (ここを固定リストにすると profile 追加時に表から静かに消える)。
compose_json() {
  local args=() profile
  while read -r profile; do
    args+=(--profile "$profile")
  done < <(docker compose config --profiles 2>/dev/null)
  docker compose "${args[@]}" config --format json 2>/dev/null
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

table="$(
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
)"

generated="$(awk -v begin="$begin_marker" -v end="$end_marker" -v table="$table" '
  index($0, begin) == 1 { print table; skip = 1; next }
  index($0, end) == 1 { skip = 0; next }
  !skip { print }
' "$readme")"

if [ "$mode" = "--write" ]; then
  printf '%s\n' "$generated" >"$readme"
  echo "service table: $readme を更新しました"
  exit 0
fi

if diff_out="$(diff -u "$readme" <(printf '%s\n' "$generated"))"; then
  echo "service table: OK"
else
  echo "README のサービス表が compose.yaml と乖離しています。scripts/docs/gen-service-table.sh --write を実行してください:" >&2
  printf '%s\n' "$diff_out" >&2
  exit 1
fi
