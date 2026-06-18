#!/usr/bin/env bash
# coverprofile から Vitest レポート風のカバレッジ Markdown を生成して stdout に出す。
# Go のカバレッジは statement 単位なので Lines/Branches は出せない。Statements (件数つき) と
# go tool cover -func 由来の Functions、さらにファイル別内訳を出す。
# 使い方: coverage-report.sh <title> [profile=cover.out]
#
# 注意: -coverpkg で複数テストバイナリから同一ブロックが重複して profile に並ぶため、
# 行を素朴に合計すると二重計上になる。go tool 同様、ブロック (file:range) 単位で
# 「どれかで count>0 ならカバー」とマージしてから集計する。
set -euo pipefail

title="$1"
profile="${2:-cover.out}"
mod="github.com/rin2yh/study-architecture/"

pct() { awk -v c="$1" -v t="$2" 'BEGIN { printf "%.1f%%", (t > 0) ? c * 100 / t : 0 }'; }

read -r stmt_cov stmt_total < <(
  awk 'NR > 1 { s[$1] = $2; if ($3 > 0) c[$1] = 1 }
       END { for (b in s) { t += s[b]; if (b in c) cov += s[b] } print cov + 0, t + 0 }' "$profile"
)

# go tool cover -func は既にブロックをマージ済み。行末の % が 0 の関数を未カバーとして数える。
read -r fn_cov fn_total < <(
  go tool cover -func="$profile" | awk '
    /^total:/ { next }
    { p = $NF; sub(/%$/, "", p); ft++; if (p + 0 > 0) fc++ }
    END { print fc + 0, ft + 0 }'
)

echo "### Coverage: ${title}"
echo
echo "| Category | Percentage | Covered / Total |"
echo "| --- | ---: | ---: |"
echo "| Statements | $(pct "$stmt_cov" "$stmt_total") | ${stmt_cov} / ${stmt_total} |"
echo "| Functions | $(pct "$fn_cov" "$fn_total") | ${fn_cov} / ${fn_total} |"
echo
echo "<details><summary>File Coverage</summary>"
echo
echo "| File | Statements | Covered / Total |"
echo "| --- | ---: | ---: |"
awk -v mod="$mod" 'NR > 1 {
  if (!($1 in s)) {
    s[$1] = $2; split($1, a, ":"); fof[$1] = a[1]
    if (!(a[1] in seen)) { seen[a[1]] = 1; ord[++n] = a[1] }
  }
  if ($3 > 0) c[$1] = 1
}
END {
  for (b in s) { f = fof[b]; ft[f] += s[b]; if (b in c) fc[f] += s[b] }
  for (i = 1; i <= n; i++) {
    f = ord[i]; name = f; sub("^" mod, "", name)
    p = (ft[f] > 0) ? fc[f] * 100 / ft[f] : 0
    printf "| %s | %.1f%% | %d / %d |\n", name, p, fc[f] + 0, ft[f]
  }
}' "$profile"
echo
echo "</details>"
