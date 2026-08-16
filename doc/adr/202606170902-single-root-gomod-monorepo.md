# ADR-202606170902: 単一ルート go.mod のモノレポ構成

- Status: Accepted
- Date: 2026-06-17

## Context

個別デプロイ可能な 5 つの Go サービスを 1 リポジトリに持つ。各サービスは `internal/` で閉じたい。
選択肢は「単一ルート go.mod」か「サービスごと独立 go.mod + go.work」。

## Decision

**単一ルート go.mod** を採用し、`go.work` は持たない。

- 各サービスは `internal/` をサービス配下に置く。Go の internal 規則により他サービスからの import が
  **コンパイル時に禁止**され、「閉じる」を言語機能で強制できる。
- 各サービスは単一コマンドなので `main.go` をサービス直下に置く（`cmd/` ネストは設けない。
  go.dev/doc/modules/layout の「単一コマンド + 補助パッケージ」に倣う）。
- 個別デプロイは各サービスの Dockerfile が自分のパッケージだけをビルドして実現する
  （context はリポジトリルート）。

## Consequences

- go.mod/go.sum が 1 組で依存管理・CI・Docker ビルドが単純（`go.work` の multi-module ビルド問題を回避）。
- 全サービスが同一の依存バージョン集合を共有する（サービス別ピンは不可）。共有最小・少数サービスの
  本プロジェクトでは実害は小さい。
- 将来サービスを別リポジトリ / 独立モジュールへ切り出す場合は、その時点で当該サービスを別 go.mod へ
  分離する（DB 分割と歩調を合わせられる）。

## Alternatives considered

- **独立 go.mod + go.work**: サービス別に依存を固定でき将来の切り出しに有利だが、Docker ビルドで
  `GOWORK=off` 等の回避が必要になり、現段階では過剰な複雑さ。
