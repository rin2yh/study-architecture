#!/usr/bin/env bash
# .claude/settings.json の extraKnownMarketplaces / enabledPlugins だけでは、
# 外部 (GitHub) ソースのプラグインはコンテナ起動時に取得されない
# (計測: 空の ~/.claude でセッションを起こしても marketplace 未登録のまま)。
# cc web は毎回まっさらなコンテナなので、SessionStart で取得を明示的に走らせる。
set -uo pipefail

MARKETPLACE_NAME="rin2yh-plugins"
MARKETPLACE_REPO="rin2yh/claude-code-plugins"
PLUGINS=(development-skills meta-skills general-skills fav-rules)

log() { echo "[install-plugins] $*" >&2; }

if ! command -v claude >/dev/null 2>&1; then
  log "claude CLI が PATH にないため skip"
  exit 0
fi

if claude plugin marketplace list 2>/dev/null | grep -q "$MARKETPLACE_NAME"; then
  log "marketplace $MARKETPLACE_NAME は登録済み"
else
  if ! claude plugin marketplace add "$MARKETPLACE_REPO" >/dev/null 2>&1; then
    log "marketplace $MARKETPLACE_REPO の追加に失敗 (ネットワーク未疎通の可能性)"
    exit 0
  fi
  log "marketplace $MARKETPLACE_NAME を追加"
fi

installed="$(claude plugin list 2>/dev/null || true)"
for plugin in "${PLUGINS[@]}"; do
  if grep -q "${plugin}@${MARKETPLACE_NAME}" <<<"$installed"; then
    continue
  fi
  if claude plugin install "${plugin}@${MARKETPLACE_NAME}" --scope project >/dev/null 2>&1; then
    log "installed ${plugin}@${MARKETPLACE_NAME}"
  else
    log "install 失敗: ${plugin}@${MARKETPLACE_NAME}"
  fi
done

exit 0
