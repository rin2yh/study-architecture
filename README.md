# EC サイト — サービスベースアーキテクチャ練習

『アーキテクチャの基礎』のサービスベースアーキテクチャを題材に、ローカル完結・費用ゼロで
EC サイトを段階的に育てる学習プロジェクト。バックエンドは Go、コード生成中心。

- アーキテクチャ方針: [doc/adr/0001](doc/adr/0001-service-based-architecture.md)
- 技術スタック: [doc/adr/0002](doc/adr/0002-codegen-first-tech-stack.md)
- リポジトリ構成: [doc/adr/0003](doc/adr/0003-single-root-gomod-monorepo.md)
- データ戦略: [doc/adr/0004](doc/adr/0004-shared-postgres-schema-per-domain.md)
- 横断データ/スナップショット: [doc/adr/0008](doc/adr/0008-cross-domain-snapshot.md)
- フロントエンド: [doc/adr/0005](doc/adr/0005-frontend-tanstack-start.md)
- UI データ取得: [doc/adr/0006](doc/adr/0006-ui-server-loader-data-fetching.md)
- UI モノレポ/命名: [doc/adr/0007](doc/adr/0007-frontend-pnpm-monorepo-tooling.md)
- 認証/セッション: [doc/adr/0009](doc/adr/0009-member-auth-httponly-cookie-session.md)
- 既知問題: [doc/known-issues.md](doc/known-issues.md)

## 構成（Step 0）

ドメインサービス 5 つ（各 1 コンテナ・個別デプロイ）。出発点は分割UI（後続）+ 直接呼び出し +
共有 Postgres（schema 分離）。

| 区分 | 名前 | ホストポート | コンテナ内 | DB schema |
| --- | --- | --- | --- | --- |
| サービス | product | 8001 | 8080 | `product` |
| サービス | order | 8002 | 8080 | `"order"` |
| サービス | payment | 8003 | 8080 | `payment` |
| サービス | member | 8004 | 8080 | `member` |
| サービス | shipping | 8005 | 8080 | `shipping` |
| データ | db (Postgres 17) | 5432 | 5432 | - |
| UI | store | 5173 | 3000 | → product / order / payment |
| UI | mypage | 5174 | 3000 | → member / order / shipping |
| UI | backoffice | 5175 | 3000 | → product / order / shipping |

各サービスは `GET /healthz`（liveness）に加え、一覧 / 取得 / 作成 / 更新を持つ
（例: `GET`/`POST /products`、`GET`/`PUT /products/{id}`）。エラー整形と 404/409/422 の扱いは
[doc/adr/0014](doc/adr/0014-api-error-model.md)、更新 (PUT) の項目方針（業務項目のみ置換・FK 不変）は
[doc/adr/0015](doc/adr/0015-update-endpoint-put-semantics.md)。
order はさらに `POST /checkout` を持ち、カート（`productId` + `quantity`）を確定して product を
参照し商品名・単価を注文明細にスナップショットしたうえで payment を手配する（配送は決済確定後に
運用系で手配する想定で同期 checkout には含めない）。横断データの扱いは
[doc/adr/0008](doc/adr/0008-cross-domain-snapshot.md)。
UI は TanStack Start。サーバ側ローダから orval 生成クライアント(+zod)で各サービスを呼ぶ（[doc/adr/0006](doc/adr/0006-ui-server-loader-data-fetching.md)）。

## 前提ツール

`mise` がツールを固定する。コード生成ツール（oapi-codegen / kessoku / sqlc / goose）は
go.mod の `tool` ディレクティブで管理し `go tool` で実行する。

```sh
mise install        # go / node を固定インストール
mise trust          # 初回のみ。go が mise shim 経由のため未trustだと codegen が失敗する
```

- Go 1.26 / Docker（compose v2）が必要。

## クイックスタート

```sh
# 1. コード生成（sqlc → oapi-codegen → kessoku）
mise gen

# 2. ビルド & テスト
mise build
mise test

# 3. 起動（db + 5サービス）
mise up

# 4. マイグレーション適用（ホストの goose、または compose のワンショット）
mise migrate
#   または: docker compose run --rm migrate up

# 動作確認（ブラウザ/HTTPクライアントで）
#   http://localhost:8001/healthz
#   http://localhost:8001/products
```

## コード生成のしくみ

契約（OpenAPI）とスキーマ（SQL）を単一情報源とし、実装コードを生成する。

```
server/<svc>/api/openapi.yaml ──oapi-codegen──▶ api/api.gen.go (型 + StrictServerInterface)
db/migration/*.sql ─┐
server/<svc>/db/query/*.sql ─┴─sqlc──▶ internal/db/*.go (型 + Querier)
internal/di/inject.go ──kessoku──▶ internal/di/inject_band.go (InitHandler)
```

手書きするのは `internal/handler`（業務ロジック）、`internal/repository`、`server/<svc>/main.go` のみ。
生成順序は依存関係上 **sqlc → oapi-codegen → kessoku**（`mise gen` が順序実行）。

## ディレクトリ

```
db/migration/             # goose 連番マイグレーション（中央集約 / schema 修飾）
server/<svc>/             # バックエンド (Go)。単一ルート go.mod、各サービスは単一コマンド
  main.go                 # package main（cmd/ ネストは置かない）
  api/                    # openapi.yaml, oapi-codegen.yaml, api.gen.go
  db/query/               # sqlc 入力クエリ
  internal/{db,repository,handler,di}/
  sqlc.yaml
  Dockerfile
client/                   # フロントエンド (pnpm workspace)
  pnpm-workspace.yaml     # packages / catalog（共通依存の一元管理） / minimumReleaseAge
  app/<app>/              # TanStack Start (store / mypage / backoffice)
    src/routes/           # ファイルベースルーティング（__root.tsx, index.tsx ...）
    vite.config.ts
  package/api/            # 共有パッケージ @ec/api
    orval.config.ts       # OpenAPI → client + zod（全サービス分）
    src/                  # 生成: <svc>/ クライアント・model・zod, mutator.ts, バレル
  Dockerfile              # 3 UI 共通（APP 引数で切替）
compose.yaml  mise.toml  Dockerfile.migrate
doc/adr/                  # 設計判断
```

## UI（TanStack Start / pnpm workspace）

`client/` は pnpm workspace。共有パッケージ `@ec/api`（orval 生成クライアント + zod + mutator）を
3 つの app（store / mypage / backoffice）が参照する。各 app の `src/routes/index.tsx` は
`createServerFn`（サーバ実行）→ `@ec/api` の fetch クライアント → zod 検証 → 描画。サービス URL は
サーバ側 env で注入（[doc/adr/0006](doc/adr/0006-ui-server-loader-data-fetching.md)）。lint/format は
**oxlint / oxfmt**。依存は **pnpm**（共通依存は **catalog** で一元管理、`minimumReleaseAge` で
公開1週間未満の版は使わない）。詳細は [doc/adr/0007](doc/adr/0007-frontend-pnpm-monorepo-tooling.md)。

```sh
cd client
pnpm install
pnpm --filter @ec/api gen   # server/<svc>/api/openapi.yaml → package/api/src/**（client + zod）
pnpm -r build               # 各 app を vite build → dist/server/server.js
pnpm -r typecheck   # tsc --noEmit
pnpm lint           # oxlint
pnpm format         # oxfmt
```

リポジトリ全体では `mise ui:install` / `mise ui:gen` / `mise ui:build` でも操作できる。

## ロードマップ

- **Step 0（現在）**: 分割UI + 直接呼び出し + 共有DB（schema 分離）
- **Step 1**: API ファサード（UI の BFF）を足す
- **Step 2**: データ所有権を確定し schema 分離を徹底
- **Step 3**: 結合の弱い縁から DB を分割（payment / shipping → member → product × order）
