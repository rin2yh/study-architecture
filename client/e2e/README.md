# e2e

Playwright による E2E。スコープは現状 `store` の購入フローシナリオのみ（Issue #10 のステップ 1）。

`store` は React Router の SSR アプリで、loader（サーバ側）が各バックエンドへ fetch する。
ブラウザ側の route mock では SSR の fetch を差し替えられないため、**compose のフルスタックに
対して** E2E を実行する。

## 前提

- `store` とバックエンド一式が起動済みであること（`baseURL` 既定 `http://localhost:5173`）。
- product サービスが host から叩けること（シード投入用、既定 `http://localhost:8001`）。

`tests/setup/seed.ts`（globalSetup）が、product のテーブルにシードが無い前提で E2E が必要と
する商品を冪等に投入する。

## 構成

テストは Page Object Model (POM) で書く。画面ごとのロケータと操作を `tests/pages/*` に閉じ込め、
spec (`tests/store/*.spec.ts`) はシナリオだけを記述する。セレクタが変わっても spec ではなく
Page Object 側だけを直せばよいようにする。

- `tests/pages/` — 画面ごとの Page Object（`HomePage` / `CartPage` / `CheckoutPage`）
- `tests/store/` — シナリオ (spec)
- `tests/setup/` — globalSetup（シード）

## ローカル実行

リポジトリルートから:

```sh
mise run test:e2e
```

スタックを自分で起動してテストだけ回す場合（`client/` 配下）:

```sh
pnpm -F e2e exec playwright install --with-deps chromium  # 初回のみ
pnpm -F e2e test
```

## 環境変数

| 変数                  | 既定                    | 用途                            |
| --------------------- | ----------------------- | ------------------------------- |
| `E2E_BASE_URL`        | `http://localhost:5173` | store の URL                    |
| `E2E_PRODUCT_API_URL` | `http://localhost:8001` | シード投入先の product サービス |
