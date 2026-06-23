# e2e

`store`（社外）と `backoffice`（社内）は到達できるネットワークが異なるため、別スタックの compose を
起動して E2E を回す。

スタックの起動は Playwright の外（`scripts/e2e-up.sh`）に置き、テスト前に mise タスク / CI から呼ぶ。
Playwright の `webServer` は project 単位に持てず、`--project` を絞っても全 webServer が起動して
しまうため。
