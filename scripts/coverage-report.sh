#!/usr/bin/env bash
# Go のカバレッジは statement 単位なので Lines/Branches は出せない。
# また -coverpkg では複数テストバイナリから同一ブロックが profile に重複して並ぶため、
# 行を素朴に合計すると二重計上になる。ブロック (file:range) 単位でマージしてから集計する。
set -euo pipefail

title="$1"
profile="${2:-cover.out}"
mod="github.com/rin2yh/study-architecture/"

# CI の coverage gate と同じ 60% を緑/赤の境にして、コメント上でも gate 結果を読み取れるようにする。
gate=60

pct() { awk -v c="$1" -v t="$2" 'BEGIN { printf "%.1f%%", (t > 0) ? c * 100 / t : 0 }'; }
icon() { awk -v c="$1" -v t="$2" -v g="$gate" 'BEGIN { p = (t > 0) ? c * 100 / t : 0; print (p >= g) ? "🟢" : "🔴" }'; }

read -r stmt_cov stmt_total < <(
  awk 'NR > 1 { s[$1] = $2; if ($3 > 0) c[$1] = 1 }
       END { for (b in s) { t += s[b]; if (b in c) cov += s[b] } print cov + 0, t + 0 }' "$profile"
)

# go tool cover -func は集計済みなので、ここは重複排除せずそのまま数える。
read -r fn_cov fn_total < <(
  go tool cover -func="$profile" | awk '
    /^total:/ { next }
    { p = $NF; sub(/%$/, "", p); ft++; if (p + 0 > 0) fc++ }
    END { print fc + 0, ft + 0 }'
)

echo "## Coverage Report for ${title}"
echo
echo "| Status | Category | Percentage | Covered / Total |"
echo "| :---: | --- | ---: | ---: |"
echo "| $(icon "$stmt_cov" "$stmt_total") | Statements | $(pct "$stmt_cov" "$stmt_total") | ${stmt_cov} / ${stmt_total} |"
echo "| $(icon "$fn_cov" "$fn_total") | Functions | $(pct "$fn_cov" "$fn_total") | ${fn_cov} / ${fn_total} |"
echo
echo "<details><summary>File Coverage</summary>"
echo
echo "| Status | File | Statements | Covered / Total |"
echo "| :---: | --- | ---: | ---: |"
awk -v mod="$mod" -v g="$gate" 'NR > 1 {
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
    printf "| %s | %s | %.1f%% | %d / %d |\n", (p >= g) ? "🟢" : "🔴", name, p, fc[f] + 0, ft[f]
  }
}' "$profile"
echo
echo "</details>"

# GitHub Actions 上でだけ Vitest と同じ出典フッタを付ける (ローカル実行では env が無く skip)。
if [[ -n "${GITHUB_RUN_ID:-}" ]]; then
  run_url="${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"
  # PR では GITHUB_SHA が merge commit になるため、workflow から head sha を COMMIT_SHA で渡す。
  sha="${COMMIT_SHA:-${GITHUB_SHA:-}}"
  echo
  if [[ -n "$sha" ]]; then
    commit_url="${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/commit/${sha}"
    echo "*Generated in workflow [#${GITHUB_RUN_NUMBER}](${run_url}) for commit [${sha:0:7}](${commit_url}) by \`scripts/coverage-report.sh\`*"
  else
    echo "*Generated in workflow [#${GITHUB_RUN_NUMBER}](${run_url}) by \`scripts/coverage-report.sh\`*"
  fi
fi
