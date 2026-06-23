# e2e

Playwright による E2E。スコープは `store` の購入フローと `backoffice` の商品管理シナリオ
（Issue #10）。

`store` / `backoffice` はいずれも React Router の SSR アプリで、loader（サーバ側）が各バックエンドへ
fetch する。ブラウザ側の route mock では SSR の fetch を差し替えられないため、**compose の
フルスタックに対して** E2E を実行する。`store` は社外スタック（profile `external`、port 5173）、
`backoffice` は社内スタック（profile `internal`、port 5175）として起動する。

## project とスタックの起動

store / backoffice は別スタックなので、Playwright の project を分けている。各 project は
スタックを起動する setup project（`store-setup` / `backoffice-setup`）に `dependencies` で
依存し、停止は setup の `teardown` project が担う。`--project=store` を指定すると Playwright は
依存する `store-setup` だけ走らせるので、起動するスタックも自然にその project に絞られる。

| project      | profile    | port | baseURL                   |
| ------------ | ---------- | ---- | ------------------------- |
| `store`      | `external` | 5173 | `E2E_BASE_URL`            |
| `backoffice` | `internal` | 5175 | `E2E_BACKOFFICE_BASE_URL` |

setup project が `scripts/e2e-up.sh <store｜backoffice>` で DB 起動 → migrate → 対象スタックの
build/up（detached）まで行い、product にシード（store はログイン用 member も）を投入し、frontend は
healthcheck を持たないため `baseURL` の到達を待ってから tests に渡す。teardown project は
`scripts/e2e-down.sh` でスタックを停止する。シードは冪等で、store のテストは `/login` 画面から
ログインする。

スタックは 1 つずつ起動する前提なので、起動は必ず `--project`（または下の mise タスク）で 1 つに
絞る。`--project` 無しで両方を 1 度に走らせると、片方の teardown が共有バックエンドを落として
もう片方を壊しうる。

## 構成

テストは Page Object Model (POM) で書く。画面ごとのロケータと操作を `tests/pages/*` に閉じ込め、
spec (`tests/store/*.spec.ts`) はシナリオだけを記述する。セレクタが変わっても spec ではなく
Page Object 側だけを直せばよいようにする。

- `tests/pages/` — 画面ごとの Page Object（store: `LoginPage` / `HomePage` / `CartPage` /
  `CheckoutPage`、backoffice: `BackofficeProductPage`）
- `tests/store/` — store のシナリオ (spec)
- `tests/backoffice/` — backoffice のシナリオ (spec)
- `tests/stack/` — setup/teardown project（compose の起動/停止と URL 到達待ち）
- `tests/setup/` — シード（商品・member）

## ローカル実行

リポジトリルートから:

```sh
mise run test:e2e             # store → backoffice を順に実行
mise run test:e2e:store       # store だけ
mise run test:e2e:backoffice  # backoffice だけ
```

`client/` 配下から直接（初回のみ `pnpm -F e2e exec playwright install --with-deps chromium`）:

```sh
pnpm -F e2e test --project=store
pnpm -F e2e test --project=backoffice
```

## 環境変数

| 変数                      | 既定                    | 用途                                        |
| ------------------------- | ----------------------- | ------------------------------------------- |
| `E2E_BASE_URL`            | `http://localhost:5173` | store の baseURL                            |
| `E2E_BACKOFFICE_BASE_URL` | `http://localhost:5175` | backoffice の baseURL                       |
| `E2E_PRODUCT_API_URL`     | `http://localhost:8001` | シード投入先の product サービス             |
| `E2E_MEMBER_API_URL`      | `http://localhost:8004` | member シード先（ログイン用アカウント作成） |
