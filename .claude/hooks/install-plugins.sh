#!/usr/bin/env bash
# settings.json に extraKnownMarketplaces / enabledPlugins を宣言してもプラグインの実体は
# 取得されない (計測: 空の ~/.claude でセッションを起こしても未取得のまま。marketplace add
# だけでも足りず plugin install が要る)。cc web は毎回まっさらなコンテナなので明示的に走らせる。
# なお cc web はプラグインのロードが SessionStart より先なのでフックが常に間に合わない。
# 初回から効かせるには Cloud environment の setup script にこのスクリプトを登録する。
claude plugin marketplace add rin2yh/claude-code-plugins

for plugin in development-skills meta-skills general-skills fav-rules; do
  claude plugin install "${plugin}@rin2yh-plugins" --scope project
done

# プラグインが無くてもリポジトリの作業自体は続けられるので、取得失敗は縮退扱いにする
# (失敗の内容は claude CLI の出力に残る)。
exit 0
