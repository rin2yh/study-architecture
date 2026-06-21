# ADR-202606170902: 単一ルート go.mod のモノレポ構成

- Status: Accepted
- Date: 2026-06-17

## Context

個別デプロイ可能な 5 つの Go サービスを 1 リポジトリに持つ。各サービスは `internal/` で
閉じたい。構成の選択肢は「単一ルート go.mod」か「サービスごと独立 go.mod + go.work」。

## Decision

**単一ルート go.mod** を採用し、`go.work` は持たない。

```
study-architecture/
├── go.mod / go.sum            # ルート1個。module github.com/rin2yh/study-architecture
├── db/migration/             # 中央集約マイグレーション（goose, schema 修飾）
├── server/<svc>/
│   ├── api/                   # openapi.yaml + oapi-codegen 生成
│   ├── db/query/            # sqlc 入力クエリ
│   ├── internal/{db,repository,handler,di}/   # ここで閉じる（他サービスから import 不可）
│   ├── main.go                # package main（単一コマンド。cmd/ ネストは置かない）
│   ├── sqlc.yaml
│   └── Dockerfile             # context=リポジトリルート
└── compose.yaml / mise.toml
```

各サービスは単一コマンドなので、go.dev/doc/modules/layout の「単一コマンド＋補助パッケージ」に倣い
`main.go` をサービス直下に置く（`cmd/` ネストは設けない）。

- 各サービスは `internal/` をサービス配下に置くことで、Go の internal 規則により
  他サービスからの import がコンパイル時に禁止される（「閉じる」を言語機能で強制）。
- 個別デプロイは各サービスの Dockerfile が自分のパッケージだけをビルドして実現。
  context はリポジトリルートで `go build ./server/<svc>`。

## Consequences

- go.mod/go.sum が 1 組で依存管理・CI・Docker ビルドが単純（`go.work` の
  multi-module ビルド問題を回避）。
- 全サービスが同一の依存バージョン集合を共有する（サービス別ピンは不可）。
  共有最小・少数サービスの本プロジェクトでは実害は小さい。
- 将来サービスを別リポジトリ/独立モジュールへ切り出す場合は、その時点で
  当該サービスを別 go.mod に分離する（Step 3 の DB 分割と歩調を合わせられる）。

## Alternatives considered

- **独立 go.mod + go.work**: サービス別に依存を固定でき将来の切り出しに有利だが、
  Docker ビルドで `GOWORK=off` 等の回避が必要になり、現段階では過剰な複雑さ。
