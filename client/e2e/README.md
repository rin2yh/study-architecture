# e2e

Playwright による store / backoffice の E2E テスト。compose のフルスタックを起動して実行する。

## 必要なもの

- docker / docker compose（スタック起動用）
- mise（タスク実行用）

## 実行

リポジトリルートから。スタックの起動からテストまでタスクが行う（初回は Playwright ブラウザの取得も走る）:

```sh
mise run test:e2e             # store → backoffice を順に実行
mise run test:e2e:store       # store のみ
mise run test:e2e:backoffice  # backoffice のみ
```
