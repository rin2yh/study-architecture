# ADR-202606170906: フロントエンドは pnpm モノレポ + oxlint/oxfmt、命名は client/server・単数

- Status: Accepted
- Date: 2026-06-17

## Context

UI をドメインごとに 3 つ（store / mypage / backoffice）へ分割した（ADR-[[202606170904]]）が、当初は各 UI を
独立した npm プロジェクトにしたため `node_modules`・設定・Dockerfile が 3 重になった。分割（個別デプロイ）
自体は維持したまま重複を排したい。あわせてリポジトリの命名規約を整える。

## Decision

### Directory naming

- 役割で二分: **`client/`（フロントエンド）** と **`server/`（バックエンド Go）**（client-server モデル）。
- **ディレクトリ名は単数**（`app/`, `db/migration/`, `doc/` 等）。

### Frontend structure

- `client/` は **pnpm workspace**。共有パッケージ `@ec/api`（orval 生成のクライアント + zod + mutator）を
  各 app が `workspace:*` で参照し、app 側はルートとアプリシェルだけを持つ。
- Docker は **単一の `client/Dockerfile`**（引数で対象 app を切替）。
- **lint/format は oxlint + oxfmt**（eslint/prettier は使わない）。Rust 製で速く、設定が単純。
- 依存管理は **pnpm**。決め手は次の 3 つ:
  - **共通依存は catalog で一元管理**し、バージョンを 1 箇所に集約する。
  - **`minimumReleaseAge = 10080`（1 週間）** で、公開直後のバージョンを使わない（サプライチェーン対策）。
  - **依存のビルドスクリプトは原則 deny**。esbuild はプリビルドバイナリで動き postinstall 不要と検証済み。

## Consequences

- 依存はストア共有 + 単一 lockfile で重複が消え、設定・Dockerfile も 1 つに集約される。
  app の分割（個別デプロイ）は維持される。
- 共有 `@ec/api` を各 app が再利用するため、OpenAPI 変更時の再生成が 1 箇所で済む。
- `minimumReleaseAge` により最新版の採用は 1 週間遅れる。
- `services/` → `server/` のリネームに伴い Go の import パスを一括更新した（module パスは不変）。

## Alternatives considered

- **独立した npm プロジェクトを app ごとに持つ**: 構成は単純だが `node_modules`・設定・Dockerfile が
  多重で管理が重い（これを本 ADR で解消）。
- **単一 UI アプリに集約**: 最も管理は楽だが、UI の個別デプロイ（ADR-[[202606170904]]）を捨てることになる。
- **eslint + prettier**: 既存資産は多いが、oxlint/oxfmt の速度と設定の単純さを優先。
