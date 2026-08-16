# EC サイト — サービスベースアーキテクチャ練習

『アーキテクチャの基礎』のサービスベースアーキテクチャを題材に、ローカル完結・費用ゼロで
EC サイトを段階的に育てる学習プロジェクト。バックエンドは Go、コード生成中心。

単一事業者の EC（storefront）を前提とし、複数出店者のマーケットプレイスではない。

ドキュメントは [doc/](doc/README.md) に集約している。

## 前提ツール

Go / Docker（compose v2）が要る。ツールのバージョンは `mise` が固定する（`mise.toml` はルート・
`server/`・`client/` にあり、それぞれ goose / go・golangci-lint / node を持つ）。コード生成ツール
（oapi-codegen / kessoku / sqlc / goose）は go.mod の `tool` ディレクティブで管理し `go tool` で実行する。

```sh
mise install                    # ルート (goose)
(cd server && mise install)     # go / golangci-lint
(cd client && mise install)     # node
mise trust                      # 初回のみ。go が mise shim 経由のため未 trust だと codegen が失敗する
```

## クイックスタート

立ち上げは順序に意味がある。以降のタスク（テスト・E2E・可観測性スタックなど）は `mise tasks` で
一覧できる（`mise.toml` のあるディレクトリごとに分かれている）。

```sh
# 1. コード生成 → ビルド
(cd server && mise run gen && mise run build)
(cd client && mise run install && mise run gen && mise run build)

# 2. DB 起動 → マイグレーション → ロール権限付与（この順序が必須。ADR-[[202606231000]]）
mise run up:db
mise run migrate
mise run grant

# 3. サービス起動（ドメインサービス 6 + ワーカー 2 + UI 2）
mise run up

# 動作確認（ホストポートは compose.yaml の ports: を参照）
curl http://localhost:8001/healthz
```

可観測性スタック（`mise run up:obs`。ADR-[[202606241356]]）は `observability` profile に隔離してあり、
`mise run up` / e2e では起動しない。未起動でもアプリは graceful degradation で動く。
