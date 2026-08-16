# フロントエンド

`client/` の構成と方針をまとめる。役割分担: 設計判断は [ADR](adr/README.md)、インストールと
コマンドは [client/README](../client/README.md)、サービス構成は [architecture](architecture.md)。

## 構成

`client/` は pnpm workspace（packages は `app/*` と `e2e`）。

- `app/store` — 顧客向けストア。買い物に加えログイン・注文履歴も担う
  （[ADR-202606221300](adr/202606221300-merge-mypage-into-store.md)）。到達先は edge-proxy 経由。
- `app/backoffice` — 運営の管理画面。運用系・顧客系のサービスへ直接到達する
  （[ADR-202606170909](adr/202606170909-split-customer-and-ops-db.md)）。
- `app/api`（パッケージ名 `api`）— 共有パッケージ。orval で各サービスの OpenAPI から fetch
  クライアントと zod を生成する。
- `app/ui`（パッケージ名 `ui`）— 2 つの app が参照する shadcn ベースのデザインシステム。
- `e2e` — Playwright による E2E。

UI は React Router v7（[ADR-202606170908](adr/202606170908-frontend-react-router-v7.md)。
当初の TanStack Start [ADR-202606170904](adr/202606170904-frontend-tanstack-start.md) を置き換え）。

## データ取得

各 app のルートモジュールはサーバ側 loader で `api` の fetch クライアントを呼び、zod で検証してから
描画する（ブラウザは UI だけを叩くため CORS は要らない）。サービス URL はサーバ側 env で注入する
（[ADR-202606170905](adr/202606170905-ui-server-loader-data-fetching.md)）。認証コンテキストの解決も
このサーバ側が担う（[ADR-202606230930](adr/202606230930-bff-auth-context-and-trust-boundary.md)）。

## コンポーネント構成 (FSD)

各 app の `src/` は Feature-Sliced Design で層分けする（`app → pages → features → entities → shared`
の一方向依存、`widgets` は必要時のみ）。`app` 層は `root.tsx` + `routes.ts`、`pages` 層は `routes/`
そのもの（route モジュール = ページ。`loader`/`action` を持ち、ページ専用の表示コンポーネントは
`routes/<page>/components/` にコロケート）。操作は `features/`、ビジネス実体は `entities/`、ドメイン
非依存の再利用部品は `shared/` に置き、各スライスの公開境界は `index.ts` に集約する。詳細は
[ADR-202606220300](adr/202606220300-frontend-fsd-component-layering.md)。

## 生成コード (`api` パッケージ)

`app/api/orval.config.ts` が単一情報源。`server/<svc>/api/openapi.yaml` から `src/gen/<svc>/` へ
fetch クライアント・model・zod を生成し、`src/mutator.ts` がサーバ側 env から baseURL を注入する。
生成物の置き場と公開境界は [ADR-202606190901](adr/202606190901-client-generated-code-layout.md)。

## デザインシステム (`ui` パッケージ)

ビルド成果物は出さず TS/CSS ソースを `exports` で直接公開する（`api` パッケージと同方針。
[ADR-202606190901](adr/202606190901-client-generated-code-layout.md)）。各 app の Vite/Tailwind が
ソースを解決・バンドルする。

- `src/components/*` — shadcn の UI キット + 共有 `page-loading`。`cn` は `../lib/utils` からの相対
  import で参照する（consumer 側の `@/` エイリアスに依存しないため）。
- `src/lib/utils.ts` — `cn`。
- `src/styles/theme.css` — Tailwind v4 のテーマトークン（`:root` / `.dark` / `@theme inline`）と base
  レイヤー。`@source "../components"` で自パッケージのコンポーネントを Tailwind のコンテンツ検出に
  登録する。

`npx shadcn@latest add <name>` は `@/lib/utils` 形式の import を生成するため、追加後に `cn` の import を
`../lib/utils`（相対）へ書き換え、`package.json` の `exports` に subpath を追記する。

## 依存管理とツール

lint/format は **oxlint / oxfmt**、依存は **pnpm**。共通依存は `pnpm-workspace.yaml` の **catalog** で
一元管理し（各 package は `catalog:` 参照）、**`minimumReleaseAge`**（1 週間）で公開直後の版は使わない。
ビルドスクリプトは原則 deny（`allowBuilds`）。詳細は
[ADR-202606170906](adr/202606170906-frontend-pnpm-monorepo-tooling.md)。
