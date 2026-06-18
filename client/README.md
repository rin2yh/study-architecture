# client — フロントエンド (pnpm workspace)

TanStack Start による 3 つの UI と、共有 API パッケージのモノレポ。

## 構成

- `app/store` — 顧客向けストア（product / order / payment）
- `app/mypage` — 会員マイページ（member / order / shipping）
- `app/backoffice` — 運営の管理画面（product / order / shipping）
- `package/api`（`api`）— 共有パッケージ。orval で各サービスの OpenAPI から fetch クライアント
  と zod を生成し、mutator がサーバ側 env から baseURL を注入する。

各 app の `src/routes/index.tsx` のローダが SSR 時にサーバ側で `api` 経由でサービスを呼ぶ
（ブラウザは UI のみ叩くため CORS 不要）。

## コマンド（pnpm）

```sh
pnpm install
pnpm api:gen              # ../server/<svc>/api/openapi.yaml → package/api/src/**（client + zod）
pnpm -r build            # 各 app を vite build → .output/server/index.mjs (Nitro node-server)
pnpm -r typecheck        # tsc --noEmit
pnpm lint                # oxlint
pnpm format              # oxfmt
pnpm --filter store dev  # 個別 app の dev サーバ（mypage / backoffice も同様）
```

## 依存管理

- 共通依存は `pnpm-workspace.yaml` の **catalog** で一元管理（各 package は `catalog:` 参照）。
- **`minimumReleaseAge`**（1 週間）で公開直後の版は使わない。ビルドスクリプトは原則 deny。

詳細は [../doc/adr/0007](../doc/adr/0007-frontend-pnpm-monorepo-tooling.md) /
[../doc/adr/0006](../doc/adr/0006-ui-server-loader-data-fetching.md)。Docker 起動時の既知問題は
[../doc/known-issues.md](../doc/known-issues.md)。
