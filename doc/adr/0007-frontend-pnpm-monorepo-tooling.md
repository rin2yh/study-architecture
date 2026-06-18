# ADR 0007: フロントエンドは pnpm モノレポ + oxlint/oxfmt、命名は client/server・単数

- Status: Accepted
- Date: 2026-06-17

## Context

UI をドメインごとに 3 つ（store / mypage / backoffice）に分割する（[[0005]]）。当初は各 UI を
独立した npm プロジェクトにしたが、`node_modules`・設定・Dockerfile が 3 重になり管理が重い。
分割（個別デプロイ）自体は維持したまま、重複を排したい。あわせてリポジトリの命名規約を整える。

## Decision

### Directory naming

- 役割で二分: **`client/`（フロントエンド）** と **`server/`（バックエンド Go）**（client-server モデル）。
- **ディレクトリ名は単数**（複数形を使わない）: `app/`, `package/`, `db/migration/`,
  `server/<svc>/db/query/`, `doc/` 等。

### Frontend structure

- `client/` は **pnpm workspace**。
  - `client/package/api`（`@ec/api`）= 共有パッケージ。orval で全サービスの OpenAPI から
    fetch クライアント + zod を生成し、mutator（サービス別 baseURL 注入）とバレルを持つ。
  - `client/app/{store,mypage,backoffice}` = 各 app。`@ec/api` を `workspace:*` で参照し、
    ルートとアプリシェルだけを持つ。
  - Docker は **単一の `client/Dockerfile`**（`APP` 引数で対象 app を切替）。Nitro の
    `.output` は自己完結のため runtime は `.output` のみ。
- **lint/format は oxlint + oxfmt**（eslint/prettier は使わない）。設定は `client/` に集約。
- 依存管理は **pnpm**。
  - **共通依存は catalog で一元管理**（`pnpm-workspace.yaml` の `catalog:`）。各 package は
    `"react": "catalog:"` のように参照し、バージョンは 1 箇所で管理する。
  - `minimumReleaseAge = 10080`（1 週間）で、公開から 1 週間未満の新しすぎるバージョンは
    使わない（サプライチェーン対策）。採用版は例として oxlint 1.69 / oxfmt 0.54 / orval 8.16。
  - 依存のビルドスクリプトは原則 **deny**（`allowBuilds: { esbuild: false }`）。esbuild は
    プリビルドバイナリ（optionalDependencies）で動くため postinstall は不要と検証済み。

## Consequences

- 依存はストア共有 + 単一 lockfile で重複が消え、設定・Dockerfile も 1 つに集約される。
  3 app の分割（個別デプロイ）は維持される。
- 共有 `@ec/api` の生成物を 3 app が再利用するため、OpenAPI 変更時の再生成が 1 箇所で済む。
- oxlint/oxfmt は Rust 製で高速。`minimumReleaseAge` により最新版は 1 週間遅れて採用される
  （例: 採用時点で oxlint 1.69 / oxfmt 0.54 / orval 8.16）。
- `services/` → `server/` のリネームに伴い Go の import パスを一括更新した（module パスは不変）。

## Alternatives considered

- **3 つの独立 npm プロジェクト**: 構成が単純だが `node_modules`・設定・Dockerfile が 3 重で
  管理が重い（これを本 ADR で解消）。
- **単一 UI アプリに集約**: 最も管理は楽だが、UI の個別デプロイ（[[0005]]）を捨てることになる。
- **eslint + prettier**: 既存資産は多いが、oxlint/oxfmt の速度と設定の単純さを優先。
