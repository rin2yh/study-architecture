# e2e

Playwright による E2E。スコープは `store` の購入フローと `backoffice` の商品管理シナリオ（Issue #10）。

`store` / `backoffice` はいずれも React Router の SSR アプリで、loader（サーバ側）が各バックエンドへ
fetch する。ブラウザ側の route mock では SSR の fetch を差し替えられないため、ブラウザ単体ではなく
**compose のフルスタックに対して** E2E を実行する。`store`（社外）と `backoffice`（社内）は到達できる
ネットワークが異なるため別スタックとして起動する。

スタックの起動は Playwright の外（`scripts/e2e-up.sh`）に置き、テスト前に mise タスク / CI から呼ぶ。
Playwright の `webServer` は project 単位に持てず、`--project` を絞っても全 webServer が起動して
しまうため。

テストは Page Object Model で書く（セレクタ変更を spec から隔離するため）。
