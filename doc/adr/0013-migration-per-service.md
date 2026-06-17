# ADR 0013: マイグレーションをサービスごとに分割する

- Status: Proposed
- Date: 2026-06-18
- Supersedes (一部): [[0004]] の「マイグレーション中央集約 (`db/migration/`)」前提

## Context

[[0004]] は Step 0 のシンプルさを優先して migration を `db/migration/` に中央集約していた。
[[0012]] で物理 DB を 2 つに分け、`db/migration/{customer,ops}/` まで分割したが、
それでも「1 ディレクトリに複数ドメインの table が混在する」状態は残っていた。

このまま放置すると、各サービスの schema 変更が **migration ディレクトリのオーナー**
(= DB 専任チーム) を介さないと触れない構造になりやすい。サービスベースアーキ
([[0001]]) の前提であるドメインオーナーシップに反する。

## Decision

**migration を各サービス内に置く** ことにする。配置は `server/<svc>/db/migration/`。

- 各サービスは自分の schema 作成 (`00001_init_schema.sql`) と table 定義 (`00002_*.sql`)
  を所有する。番号付けは **サービス内独立** (各サービスが 00001 から開始)。
- 物理 DB の振り分けは [[0012]] のまま:
  - **db-customer** に流す: `order` / `payment` / `member`
  - **db-ops** に流す:      `product` / `shipping`
- sqlc の入力 schema は各サービスの `sqlc.yaml` で `schema: "db/migration"` (自身) を指す。
- compose では `migrate-<svc>` のジョブを 5 つ持ち、各オーナーが
  `docker compose --profile tools run --rm migrate-<svc> up` で個別に流せる。
- mise でも `migrate:<svc>` を 5 つ、`migrate` で順次束ねる。

## Consequences

- **ドメインオーナーシップ**: 各サービスの開発者が `server/<svc>/db/migration/` を直接
  触れる。中央 DB チームを介さない。
- **PR の影響範囲が小さい**: schema 変更が 1 サービスのディレクトリに収まる。レビュアー
  も該当ドメインの担当者だけで足りる。
- **goose のバージョンテーブルが service ごとに独立**: `goose_db_version` テーブルが
  schema ごとに作られる (goose のデフォルトは `public.goose_db_version` だが、各サービスが
  自身の DB / schema を使う設計でも、現状の挙動を確認した上で問題なければそのまま採用)。
- **テーブル名の重複は schema で防ぐ**: 各サービスは自身の schema 内に table を作るので、
  サービス間でテーブル名が被っても衝突しない。
- **横断データは [[0008]] のスナップショット保存**: 横断 JOIN は依然禁止。

## Rollout

- 既存の `db/migration/{customer,ops}/` を捨て、各サービスの `server/<svc>/db/migration/`
  に移動する。
- 各サービスに `00001_init_schema.sql` (CREATE SCHEMA) と `00002_<table>.sql` (CREATE TABLE)
  を置く。
- `Dockerfile.migrate` は `server/<svc>/db/migration` を `/migrations/<svc>` に COPY する。
- `compose.yaml` の migrate ジョブを `migrate-order` / `migrate-payment` / `migrate-member` /
  `migrate-product` / `migrate-shipping` の 5 つにする。
- `server/mise.toml` の migrate task も同じく 5 つに分け、`mise run migrate` で順次束ねる。

## Alternatives considered

- **db/migration/ 中央集約のまま (現状)**: 触る人が DB 専任に偏る。サービスオーナーシップを
  保ちたい本プロジェクトの前提に合わない。
- **db/migration/<svc>/ にぶら下げる**: schema 一覧は見やすいが、サービスとファイルが
  リポジトリ上で離れる。「サービスの中を編集する流れで migration も一緒に直す」体験が
  弱くなる。
- **マイクロサービスごとに別 module / 別 repo**: 規模が出てからの選択肢。今は単一 go.mod
  (ADR 0003) を維持する。
