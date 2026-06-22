# e2e

Playwright による E2E。スコープは `store` の購入フローと `backoffice` の商品管理シナリオ
（Issue #10）。

`store` / `backoffice` はいずれも React Router の SSR アプリで、loader（サーバ側）が各バックエンドへ
fetch する。ブラウザ側の route mock では SSR の fetch を差し替えられないため、**compose の
フルスタックに対して** E2E を実行する。`store` は社外スタック（profile `external`、port 5173）、
`backoffice` は社内スタック（profile `internal`、port 5175）として起動する。

## スタックの起動

compose スタックの起動は Playwright の `webServer`（`scripts/e2e-up.sh`）が担う。テスト実行時に
DB 起動 → migrate → 社外・社内スタック build/up まで行い、`baseURL`（store）への応答を起動完了の
検知に使う。ローカルで既に起動済みなら `reuseExistingServer` で再利用する（CI では常に起動する）。

`tests/setup/seed.ts`（globalSetup）が、product のテーブルにシードが無い前提で必要な商品を冪等に
投入し、ログインに使う member も用意する。テストは store の `/login` 画面からログインする。

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
mise run test:e2e
```

`client/` 配下から直接:

```sh
pnpm -F e2e exec playwright install --with-deps chromium  # 初回のみ
pnpm -F e2e test
```

## 環境変数

| 変数                      | 既定                    | 用途                                        |
| ------------------------- | ----------------------- | ------------------------------------------- |
| `E2E_BASE_URL`            | `http://localhost:5173` | store の URL（Playwright の baseURL）       |
| `E2E_BACKOFFICE_BASE_URL` | `http://localhost:5175` | backoffice の URL                           |
| `E2E_PRODUCT_API_URL`     | `http://localhost:8001` | シード投入先の product サービス             |
| `E2E_MEMBER_API_URL`      | `http://localhost:8004` | member シード先（ログイン用アカウント作成） |
