# ADR 0002: コード生成中心の技術スタック

- Status: Accepted (Go サーバの `std-http-server` 部分は [[0010]] により Superseded)
- Date: 2026-06-17

## Context

「実装するコードそのものをなるべく生成する」方針。契約（OpenAPI）とスキーマ（SQL）を
単一情報源とし、手書きはグルー（業務ロジック）に絞りたい。バックエンドは Go、
フロントは TypeScript。バリデーションは zod、Go の DI は kessoku、ORM は SQL 中心。

## Decision

| 関心事 | ツール | バージョン | 備考 |
| --- | --- | --- | --- |
| API（Go サーバ生成） | oapi-codegen | v2.7.1 | `std-http-server` + `strict-server`。net/http(Go 1.22+ ServeMux)、FW非依存 |
| DB アクセス（Go） | sqlc | v1.31.1 | `engine: postgresql` / `sql_package: pgx/v5`。SQL から型安全コード生成 |
| マイグレーション | goose | v3.27.1 | SQL マイグレーション。goose ファイルを sqlc の schema 入力に流用（SSOT） |
| DI | kessoku | v1.1.0 | コンパイル時 DI 生成。`Async` で独立 provider を並列初期化 |
| タスク/バージョン | mise | - | go/node 固定 + タスク（gen/migrate/up/test...） |
| TS クライアント+zod | orval | v8 | OpenAPI → fetch client + zod スキーマ（後続UIで使用） |

すべての Go コード生成ツールは **go.mod の `tool` ディレクティブ**（Go 1.24+）で管理し
`go tool <name>` で実行する。バージョンが go.mod に固定され再現性が高い。

生成の依存順序は **sqlc → oapi-codegen → kessoku**。kessoku の配線対象 handler が
oapi 生成の interface を実装し、repository が sqlc 生成の Querier を使うため kessoku が最後。
`go generate -run <tool> ./...` で順序を明示実行する（`mise gen`）。

## Consequences

- OpenAPI / SQL を直せば型・サーバ・クライアント・DI が再生成され、手書き量が減る。
- 生成物（`api.gen.go` / `internal/db/*` / `inject_band.go`）は**コミット**する
  （Docker ビルドの簡素化と再現性のため）。生成は `mise gen`。
- ツール依存が go.mod に入りモジュールグラフは大きくなるが、再現性を優先する。
- kessoku の生成ファイルは `*_band.go` 命名（`*_kessoku.go` ではない）。

## Alternatives considered

- DB: GORM 等のフルORM → 「SQL 中心」方針に反する。sqlc を採用。
- API: 手書き net/http → 契約と実装の乖離が生じる。OpenAPI 駆動にする。
- DI: google/wire → kessoku は wire 系で並列初期化に対応。指定通り kessoku。
- ツール管理: 各自 `go install` → バージョンばらつき。`go tool` で固定。
