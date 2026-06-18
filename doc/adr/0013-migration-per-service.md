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
- マイグレーションは **アプリ起動と分離して** 流す。アプリ起動時の自動マイグレーションは
  しない:
  - 複数インスタンスのデプロイで競合する
  - migrate 失敗で起動できないとロールアウトのアトミック性が崩れる
  - ローリングデプロイ時に「新アプリ + 古い schema」「古いアプリ + 新 schema」の窓ができる
  - 高権限 (DDL) を全インスタンスに持たせる必要が出てしまう
- 流す場所:
  - **CI/CD パイプライン**: deploy step の手前で `go tool goose -dir server/<svc>/db/migration up`
    を流す (本番運用の本筋)
  - **ローカル開発**: `mise run migrate:<svc>` (host から goose) か、compose の
    `docker compose --profile tools run --rm migrate-<svc>` で 1 サービスずつ流す
- compose の migrate ジョブは `image: ec-migrate` を共有して build を 1 回で済ます。
- 顧客系 3 サービスが同じ `db-customer` を共有するため、goose の version 管理表は
  サービス別に **`goose_<svc>_version`** で分ける (compose / mise / CI で同じ規約)。

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
- `Dockerfile.migrate` は 5 サービスの `db/migration/` を `/migrations/<svc>` に COPY する
  単一の image を作る (`image: ec-migrate` を compose の 5 ジョブで共有して build 1 回)。
- compose の `migrate-<svc>` (顧客系は `compose.yaml`、運用系は `compose.internal.yaml`) は
  `profiles: ["tools"]` を付けて up の対象外にしておく。
- `server/mise.toml` の `migrate:<svc>` で host の goose を直接叩く逃げ道も用意する。
  `mise run migrate` で 5 サービス分を順次束ねる。
- CI の integration job では migrate step で 5 サービス分を順に流してから test を実行する。

## Alternatives considered

- **db/migration/ 中央集約のまま (現状)**: 触る人が DB 専任に偏る。サービスオーナーシップを
  保ちたい本プロジェクトの前提に合わない。
- **db/migration/<svc>/ にぶら下げる**: schema 一覧は見やすいが、サービスとファイルが
  リポジトリ上で離れる。「サービスの中を編集する流れで migration も一緒に直す」体験が
  弱くなる。
- **マイクロサービスごとに別 module / 別 repo**: 規模が出てからの選択肢。今は単一 go.mod
  (ADR 0003) を維持する。
