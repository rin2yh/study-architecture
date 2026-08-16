# ADR-202606170901: コード生成中心の技術スタック

- Status: Accepted (Go サーバの `std-http-server` 部分は ADR-[[202606170907]] により Superseded)
- Date: 2026-06-17

## Context

「実装するコードそのものをなるべく生成する」方針。契約（OpenAPI）とスキーマ（SQL）を単一情報源とし、
手書きは業務ロジックに絞りたい。バックエンドは Go、フロントは TypeScript。

## Decision

関心事ごとに生成ツールを据える。バージョンは go.mod / `mise.toml` が持つので、ここでは選定だけを残す。

- **API**: oapi-codegen（OpenAPI → Go サーバ。FW 非依存の生成を選ぶ）
- **DB アクセス**: sqlc（SQL から型安全コード生成。SQL 中心方針に合う）
- **マイグレーション**: goose。goose のファイルをそのまま sqlc の schema 入力に流用し、スキーマを SSOT にする
- **DI**: kessoku（コンパイル時 DI 生成。独立 provider の並列初期化を持つ）
- **TS クライアント + zod**: orval（OpenAPI → fetch client + zod）
- **タスク / バージョン固定**: mise

Go の生成ツールは **go.mod の `tool` ディレクティブ**で管理し `go tool <name>` で実行する。
バージョンが go.mod に固定され再現性が高い。

生成の依存順序は **sqlc → oapi-codegen → kessoku**。kessoku の配線対象 handler が oapi 生成の interface を
実装し、repository が sqlc 生成の Querier を使うため kessoku が最後になる。

## Consequences

- OpenAPI / SQL を直せば型・サーバ・クライアント・DI が再生成され、手書き量が減る。
- 生成物は**コミットする**（Docker ビルドの簡素化と再現性のため）。
- ツール依存が go.mod に入りモジュールグラフは大きくなるが、再現性を優先する。

## Alternatives considered

- **GORM 等のフル ORM**: 「SQL 中心」方針に反する。
- **手書き net/http**: 契約と実装の乖離が生じる。OpenAPI 駆動にする。
- **google/wire**: kessoku は wire 系で並列初期化に対応するため、そちらを採る。
- **各自 `go install` でのツール管理**: バージョンがばらつく。`go tool` で固定する。
