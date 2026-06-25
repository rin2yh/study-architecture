# ADR-202606180900: マイグレーションをサービスごとに分割する

- Status: Accepted
- Date: 2026-06-18
- Supersedes (一部): ADR-[[202606170903]] の「マイグレーション中央集約 (`db/migration/`)」前提

## Context

- ADR-[[202606170903]] は migration を `db/migration/` に中央集約。ADR-[[202606170909]] で
  `db/migration/{customer,ops}/` まで分けたが、1 ディレクトリに複数ドメインの table が混在する状態は残った。
- このままだと schema 変更が migration ディレクトリのオーナー (DB 専任チーム) を介する構造になり、
  サービスベースアーキテクチャ (ADR-[[202606170900]]) のドメインオーナーシップに反する。

## Decision

- migration を各サービス内 `server/<svc>/db/migration/` に置く。番号付けはサービス内独立 (各 00001 開始)。
- 物理 DB の振り分けは ADR-[[202606170909]] のまま。
- マイグレーションはアプリ起動と分離して流す (起動時自動マイグレーションはしない)。why:
  複数インスタンス起動で競合する / migrate 失敗で起動不能になるとロールアウトのアトミック性が崩れる /
  ローリングデプロイ中に「アプリと schema の新旧不一致」窓ができる / 全インスタンスに DDL 権限を持たせたくない。
- 流し方は `scripts/migrate.sh` 1 本に集約 (host の `go tool goose` を 5 サービス分順に呼ぶ。専用 container は作らない)。
- 顧客系 3 サービスが `db-customer` を共有するため、goose の version 表は `goose_<svc>_version` で分ける。

## Consequences

- 各サービス開発者が自分の `server/<svc>/db/migration/` を直接触れ、PR の影響範囲も 1 サービスに収まる。
- table 名はサービスごとの schema 内に作るため、サービス間で名前が被っても衝突しない。
- 横断 JOIN は依然禁止。横断データは ADR-[[202606190900]] のスナップショット保存で扱う。

## Alternatives considered

- **db/migration/ 中央集約のまま**: 触る人が DB 専任に偏り、サービスオーナーシップの前提に合わない。
- **db/migration/<svc>/ にぶら下げる**: schema 一覧は見やすいが、サービスとファイルがリポジトリ上で離れ、
  「サービスを編集する流れで migration も直す」体験が弱い。
- **サービスごとに別 module / 別 repo**: 規模が出てからの選択肢。今は単一 go.mod (ADR-[[202606170903]]) を維持する。
