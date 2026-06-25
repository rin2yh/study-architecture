# ADR-202606170907: Go サーバの HTTP フレームワークに Gin を採用

- Status: Accepted
- Date: 2026-06-17
- Supersedes: ADR-[[202606170901]] の「`std-http-server` / FW 非依存」部分

## Context

ADR-[[202606170901]] では oapi-codegen の `std-http-server` を採用し FW 非依存とした。
Step 0 の薄い骨格には十分だったが、続く実装で middleware (ログ / panic 復帰 / CORS /
認証 / rate limit / トレース) とルートのグループ化が要る場面が増える。標準 `ServeMux`
でも書けるが手書き量が多く、5 サービスに横展開する boilerplate も無視できない。

## Decision

**Gin (`github.com/gin-gonic/gin`) を採用**する。oapi-codegen を `gin-server` (非 strict)
で生成し、各サービスの `main.go` を gin ベースで組む。決め手:

- middleware の追加・除去が `engine.Use(...)` 1 行で済み、横展開コストが下がる。
- エラーレスポンス整形を `server/internal/middleware.ErrorJSON()` 1 箇所に集約できる
  (handler は `c.Error(err)` で積むだけ)。
- strict server は使わない。契約駆動の compile-time チェックは弱まるが、エラー整形の
  集約を優先する。必要になれば `strict-server: true` に戻せる (yaml 1 行で切替)。

## Consequences

- FW 非依存ではなくなる。framework 乗り換え時は oapi-codegen の `server` ジェネレータを
  切替えて再生成する手順が要る。
- 既存テストは `api.RegisterHandlers(engine, ...)` + `engine.ServeHTTP(...)` ベースへ
  書き換える (HTTP 駆動は `httptest.NewRecorder()` のまま)。
- Step 0 で `gin.Default()` (Logger + Recovery) を使うか `gin.New()` に必要分のみ Use
  するかは、検証が進んだら判断する。

## Alternatives considered

- **`std-http-server` を継続**: 依存ゼロのままだが middleware・ルート構造化の手書きコスト
  が 5 サービス分線形に増え、運用の現実とずれる。却下。
- **chi (`go-chi/chi`)**: 軽量で `net/http` 互換性が高いが、Gin の方がエコシステム
  (middleware・examples) が広く、学習材料としての参照価値が高い。
- **echo / fiber / iris**: 機能差は小さいが、Gin の方が事例・middleware の流通量が安定。
- **Connect / gRPC-Gateway 系**: 契約が protobuf。本プロジェクトは OpenAPI が SSOT
  (ADR-[[202606170901]]) なので合わない。
