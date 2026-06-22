# e2e

Playwright による E2E。スコープは `store` の購入フローと `backoffice` の商品管理シナリオ
（Issue #10）。

`store` / `backoffice` はいずれも React Router の SSR アプリで、loader（サーバ側）が各バックエンドへ
fetch する。ブラウザ側の route mock では SSR の fetch を差し替えられないため、**compose の
フルスタックに対して** E2E を実行する。`store` は社外スタック（profile `external`、port 5173）、
`backoffice` は社内スタック（profile `internal`、port 5175）として起動する。

## project とスタックの起動

store / backoffice は別スタックなので、Playwright の project を分けている。`E2E_PROJECT` で
起動するスタックを片方に絞れる（store=社外 / backoffice=社内）。毎回両方を動かすわけではない
ため、片側だけ動かしたいときは `--project` と `E2E_PROJECT` を合わせて指定する。

| project      | profile    | port | baseURL                   |
| ------------ | ---------- | ---- | ------------------------- |
| `store`      | `external` | 5173 | `E2E_BASE_URL`            |
| `backoffice` | `internal` | 5175 | `E2E_BACKOFFICE_BASE_URL` |

compose スタックの起動は Playwright の `webServer`（`scripts/e2e-up.sh <store\|backoffice>`）が
担う。テスト実行時に DB 起動 → migrate → 対象スタックの build/up まで行い、各 project の `baseURL`
への応答を起動完了の検知に使う。ローカルで既に起動済みなら `reuseExistingServer` で再利用する
（CI では常に起動する）。CI では `compose.ci.yaml` を重ねて docker layer を GitHub Actions cache
（buildx `type=gha`）に出し入れし、build を短縮する。

`tests/setup/seed.ts`（globalSetup）が、product のテーブルにシードが無い前提で必要な商品を冪等に
投入し、ログインに使う member も用意する。store のテストは `/login` 画面からログインする。

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

`client/` 配下から直接（初回のみ `pnpm -F e2e exec playwright install --with-deps chromium`）:

```sh
E2E_PROJECT=store pnpm -F e2e test --project=store
E2E_PROJECT=backoffice pnpm -F e2e test --project=backoffice
```

## 環境変数

| 変数                      | 既定                    | 用途                                         |
| ------------------------- | ----------------------- | -------------------------------------------- |
| `E2E_PROJECT`             | （未指定 = 両方）       | 起動スタックを絞る（`store` / `backoffice`） |
| `E2E_BASE_URL`            | `http://localhost:5173` | store の baseURL                             |
| `E2E_BACKOFFICE_BASE_URL` | `http://localhost:5175` | backoffice の baseURL                        |
| `E2E_PRODUCT_API_URL`     | `http://localhost:8001` | シード投入先の product サービス              |
| `E2E_MEMBER_API_URL`      | `http://localhost:8004` | member シード先（ログイン用アカウント作成）  |
