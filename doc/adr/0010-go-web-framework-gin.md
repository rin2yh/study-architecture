# ADR 0010: Go サーバの HTTP フレームワークに Gin を採用

- Status: Proposed
- Date: 2026-06-17
- Supersedes: [[0002]] の「`std-http-server` / FW 非依存」部分

## Context

[[0002]] では oapi-codegen の `std-http-server` (`net/http` + Go 1.22+ `ServeMux`)
を採用して FW 非依存とした。Step 0 の薄い骨格には十分だったが、続く実装で

- ミドルウェア (リクエストログ / panic 復帰 / CORS / 認証 / rate limit / トレース)
- グループ化されたルート定義 (`/v1/...` / `/admin/...`)
- 多くの middleware を組み合わせるエコシステム

が要る場面が増える。標準 `ServeMux` でも実装できるが手書き量が多く、5 サービスに
横展開する際の boilerplate も無視できない。

## Decision

**Gin (`github.com/gin-gonic/gin`) を採用**する。oapi-codegen の `gin-server` で
StrictHandler / RegisterHandlers を生成し、各サービスの `main.go` は
`gin.New()` (or `gin.Default()`) に StrictHandler を登録する形にする。

- 5 サービスの `server/<svc>/api/oapi-codegen.yaml` を `std-http-server: true`
  から `gin-server: true` に変更し `mise gen` で再生成する。
- `httperror.RequestErrorHandler` / `ResponseErrorHandler` は引き続き
  `func(w http.ResponseWriter, r *http.Request, err error)` シグネチャで利用可
  (oapi-codegen の StrictHandler が net/http 互換のためそのまま渡せる)。
- 共通 middleware (例: `gin.Recovery()`, `slog` リクエストログ, CORS) は
  サービス間で共有する必要が出れば `server/internal/ginx`(命名仮) に集約する。
  今は各 `main.go` 内で完結させる。

## Consequences

- Go 依存に `github.com/gin-gonic/gin` が加わる (mode 切替の OS 依存はない)。
- ミドルウェアの追加・除去が `engine.Use(...)` で 1 行になり、横展開コストが下がる。
- フレームワーク非依存ではなくなるため、将来 framework を乗り換える際は
  oapi-codegen の `server` ジェネレータを切替えて再生成する手順を踏む。
- `api.HandlerFromMux(...)` ベースのテストは `api.RegisterHandlers(engine, ...)` +
  `engine.ServeHTTP(...)` ベースに書き換える。テスト用の HTTP 駆動は変わらず
  `httptest.NewRecorder()` でよい。
- Gin のデフォルト `gin.Default()` は `Logger` + `Recovery` を有効化する。Step 0 では
  `gin.New()` に必要な middleware (`Recovery` のみ) を明示的に Use する選択も
  あり、検証が進んだら判断する。

## Alternatives considered

- **`std-http-server` を継続**: 依存はゼロのままだが middleware ・ルート構造化を
  各サービスで手書きするコストが線形に増える。学習目的としては素朴だが、5 サービス
  運用の現実とずれる。却下。
- **chi (`go-chi/chi`)**: `net/http` ハンドラと互換性が高く軽量。Gin と用途が近いが、
  Gin の方がエコシステム (middleware・examples) が広く、本プロジェクトの学習材料
  としての参照価値が高い。今回は Gin を選ぶ。
- **echo / fiber / iris**: 機能差は大きくないが、Gin の方が国内外含めて事例・
  middleware の流通量が安定。シンプルな選定で済む。
- **Connect / gRPC-Gateway 系**: 契約は protobuf。本プロジェクトは OpenAPI が SSOT
  ([[0002]]) なので合わない。

## 適用順序

- 各サービスの `oapi-codegen.yaml` を `gin-server: true` に変更
- `mise gen` で再生成 (sqlc → oapi-codegen → kessoku の順は変わらない)
- 各 `main.go` を gin ベースに書き換え (`api.RegisterHandlers(engine, si)`)
- 各 `handler_test.go` の `newServer` を gin ベースに調整
- `go mod tidy` で `gin` を `require` に追加
- [[0002]] の Status を Accepted のまま (Step 0 の出発点として尊重) とし、本 ADR が
  当該 framework 部分を **置き換える** (Supersedes) と明記する
