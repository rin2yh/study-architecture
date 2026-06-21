#!/usr/bin/env bash
# why: hook の判定は LLM を含み静的には確かめられないので、入力 → 実行 → exit code で実測する。
set -uo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
hook="$here/../check-what-comments.sh"
evals="$here/evals.json"
export CLAUDE_PROJECT_DIR="$(cd "$here/../../.." && pwd)"

pass=0
fail=0
n="$(jq '.evals | length' "$evals")"

printf '%-3s %-7s %-6s %-6s %s\n' "id" "det" "want" "got" "desc"
printf '%s\n' "-------------------------------------------------------------------"

for i in $(seq 0 $((n - 1))); do
  want="$(jq -r ".evals[$i].expect_exit" "$evals")"
  det="$(jq -r ".evals[$i].deterministic" "$evals")"
  desc="$(jq -r ".evals[$i].desc" "$evals")"
  payload="$(jq -c ".evals[$i].input" "$evals")"

  err="$(printf '%s' "$payload" | bash "$hook" 2>&1 1>/dev/null)"
  got=$?

  if [ "$got" = "$want" ]; then
    mark="PASS"; pass=$((pass + 1))
  else
    mark="FAIL"; fail=$((fail + 1))
  fi
  printf '%-3s %-7s %-6s %-6s %s [%s]\n' "$i" "$det" "$want" "$got" "$desc" "$mark"
  if [ "$mark" = "FAIL" ] && [ -n "$err" ]; then
    printf '      stderr: %s\n' "$(printf '%s' "$err" | head -n2 | tr '\n' ' ')"
  fi
done

printf '%s\n' "-------------------------------------------------------------------"
printf 'PASS=%d FAIL=%d / %d\n' "$pass" "$fail" "$n"
[ "$fail" = 0 ]
