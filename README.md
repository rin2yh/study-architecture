# EC サイト — サービスベースアーキテクチャ練習

『アーキテクチャの基礎』のサービスベースアーキテクチャを題材に、ローカル完結・費用ゼロで
EC サイトを段階的に育てる学習プロジェクト。バックエンドは Go、コード生成中心。

- アーキテクチャ方針: [docs/adr/0001](docs/adr/0001-service-based-architecture.md)
- 技術スタック: [docs/adr/0002](docs/adr/0002-codegen-first-tech-stack.md)
- リポジトリ構成: [docs/adr/0003](docs/adr/0003-single-root-gomod-monorepo.md)
- データ戦略: [docs/adr/0004](docs/adr/0004-shared-postgres-schema-per-domain.md)
- フロントエンド: [docs/adr/0005](docs/adr/0005-frontend-tanstack-start.md)

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

各サービスは `GET /healthz`（liveness）と一覧エンドポイント（例: `GET /products`）を持つ薄い骨格。

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
services/<svc>/api/openapi.yaml ──oapi-codegen──▶ api/api.gen.go (型 + StrictServerInterface)
db/migrations/*.sql ─┐
services/<svc>/db/queries/*.sql ─┴─sqlc──▶ internal/db/*.go (型 + Querier)
internal/di/inject.go ──kessoku──▶ internal/di/inject_band.go (InitHandler)
```

手書きするのは `internal/handler`（業務ロジック）、`internal/repository`、`cmd/<svc>/main.go` のみ。
生成順序は依存関係上 **sqlc → oapi-codegen → kessoku**（`mise gen` が順序実行）。

## ディレクトリ

```
db/migrations/            # goose 連番マイグレーション（中央集約 / schema 修飾）
services/<svc>/
  api/                    # openapi.yaml, oapi-codegen.yaml, api.gen.go
  db/queries/             # sqlc 入力クエリ
  internal/{db,repository,handler,di}/
  cmd/<svc>/main.go
  sqlc.yaml
  Dockerfile
compose.yaml  mise.toml  Dockerfile.migrate
docs/adr/                 # 設計判断
```

## ロードマップ

- **Step 0（現在）**: 分割UI + 直接呼び出し + 共有DB（schema 分離）
- **Step 1**: API ファサード（UI の BFF）を足す
- **Step 2**: データ所有権を確定し schema 分離を徹底
- **Step 3**: 結合の弱い縁から DB を分割（payment / shipping → member → product × order）
