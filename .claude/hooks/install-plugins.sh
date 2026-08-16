#!/usr/bin/env bash
# settings.json の宣言だけでは実体が取得されない (計測済み)。
# cc web はプラグインのロードが SessionStart より先なので、初回から効かせるには
# Cloud environment の setup script にも登録が要る。
claude plugin marketplace add rin2yh/claude-code-plugins

for plugin in development-skills meta-skills general-skills fav-rules; do
  claude plugin install "${plugin}@rin2yh-plugins" --scope project
done

# プラグインが無くても作業は続けられるので取得失敗は縮退させる
exit 0
