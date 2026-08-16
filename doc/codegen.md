# コード生成とディレクトリ構成

バックエンドの生成物と、リポジトリの置き場所をまとめる。役割分担: 設計判断は
[ADR](adr/README.md)、サービス構成とドメインの振る舞いは [architecture](architecture.md)、
UI 側の生成（orval）は [frontend](frontend.md)。

## コード生成のしくみ

契約（OpenAPI）とスキーマ（SQL）を単一情報源とし、実装コードを生成する
（[ADR-202606170901](adr/202606170901-codegen-first-tech-stack.md)）。

```
server/<svc>/api/openapi.yaml ──oapi-codegen──▶ api/api.gen.go (型 + gin の ServerInterface)
server/<svc>/db/migration/*.sql ─┐
server/<svc>/db/query/*.sql ─────┴─sqlc──▶ internal/db/*.go (型 + Querier)
internal/di/inject.go ──kessoku──▶ internal/di/inject_band.go (InitApp)
他サービスの openapi.yaml ──oapi-codegen──▶ internal/client/<svc>/client.gen.go (同期呼び出し用)
```

手書きするのは `internal/handler`（業務ロジック）、`internal/rdb`（永続化。
[ADR-202606190903](adr/202606190903-repository-cqrs-query-command.md)）、エントリポイント
（`main.go` または `cmd/<bin>/main.go`）。

生成は各パッケージの `go:generate` に紐づけてあり、`server/` で `mise run gen` を叩くと
`go generate ./...` → `go mod tidy` が走る。生成ツール（oapi-codegen / kessoku / sqlc / goose）は
go.mod の `tool` ディレクティブで管理し `go tool` で実行する。

## ディレクトリ

```
server/                   # バックエンド (Go)。単一ルート go.mod (ADR-[[202606170902]])
  <svc>/
    main.go               # 単一バイナリのサービス。package main
    cmd/{server,worker}/  # server + worker の 2 バイナリを持つサービス (shipping / inventory)
    api/                  # openapi.yaml, oapi-codegen.yaml, api.gen.go
    db/migration/         # goose マイグレーション (サービスごと。ADR-[[202606180900]])
    db/query/             # sqlc 入力クエリ
    internal/{db,rdb,handler,di}/
    sqlc.yaml
    Dockerfile            # 2 バイナリのサービスは docker/{server,worker}.Dockerfile
  internal/               # サービス横断の共通パッケージ (messaging / outbox / otelx / middleware ...)
client/                   # フロントエンド (pnpm workspace)。doc/frontend.md
infra/                    # edge-proxy と可観測性スタックの設定
scripts/                  # migrate / grant / e2e 起動などの補助スクリプト
doc/adr/                  # 設計判断
doc/ops/                  # 運用ランブック・ダッシュボードの見方
compose.yaml  mise.toml
```
