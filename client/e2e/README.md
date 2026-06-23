# e2e

Playwright による E2E。スコープは `store` の購入フローと `backoffice` の商品管理シナリオ
（Issue #10）。

`store` / `backoffice` はいずれも React Router の SSR アプリで、loader（サーバ側）が各バックエンドへ
fetch する。ブラウザ側の route mock では SSR の fetch を差し替えられないため、**compose の
フルスタックに対して** E2E を実行する。`store` は社外スタック（profile `external`、port 5173）、
`backoffice` は社内スタック（profile `internal`、port 5175）として起動する。

## project とスタックの起動

store / backoffice は別スタックなので、Playwright の project を分けている。Playwright は
webServer を project 単位に持てないため、スタックの起動は Playwright の外（`scripts/e2e-up.sh`）に
出し、テスト実行前に mise タスク / CI から呼ぶ。`scripts/e2e-up.sh <store｜backoffice>` が
DB 起動 → migrate → 対象スタックの build/up（detached）→ frontend の到達待ち（healthcheck が
無いため）まで行う。

| project      | profile    | port | baseURL                   |
| ------------ | ---------- | ---- | ------------------------- |
| `store`      | `external` | 5173 | `E2E_BASE_URL`            |
| `backoffice` | `internal` | 5175 | `E2E_BACKOFFICE_BASE_URL` |

シードは `tests/setup/seed.ts`（globalSetup）が担い、起動済みスタックへ product と member を冪等に
投入する（store のテストは `/login` からログインする）。停止は既存の `mise run down` を使う。

## 構成

テストは Page Object Model (POM) で書く。画面ごとのロケータと操作を `tests/pages/*` に閉じ込め、
spec (`tests/store/*.spec.ts`) はシナリオだけを記述する。セレクタが変わっても spec ではなく
Page Object 側だけを直せばよいようにする。

- `tests/pages/` — 画面ごとの Page Object（store: `LoginPage` / `HomePage` / `CartPage` /
  `CheckoutPage`、backoffice: `BackofficeProductPage`）
- `tests/store/` — store のシナリオ (spec)
- `tests/backoffice/` — backoffice のシナリオ (spec)
- `tests/setup/` — globalSetup（商品・member のシード）

## ローカル実行

リポジトリルートから:

```sh
mise run test:e2e             # store → backoffice を順に実行
mise run test:e2e:store       # store だけ
mise run test:e2e:backoffice  # backoffice だけ
```

スタック起動済みなら（`mise run test:e2e:store` の後など）、`client/` 配下から直接テストだけ再実行できる:

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
