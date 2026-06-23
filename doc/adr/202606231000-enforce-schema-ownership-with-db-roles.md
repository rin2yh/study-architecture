# ADR-202606231000: データ所有権を schema ごとの最小権限 DB ロールで強制する

- Status: Accepted
- Date: 2026-06-23
- Relates to: ADR-[[202606170903]] / ADR-[[202606170909]] / ADR-[[202606180900]] / ADR-[[202606190900]]

## Context

ロードマップの Step 2 は「データ所有権を確定し schema 分離を徹底 / 他ドメインへの書き込みは
持ち主サービス経由に寄せる」(ADR-[[202606170900]])。アプリ層では既にこれが満たされている:
各サービスの sqlc クエリは自分の schema しか触らず、横断データは order が product を API 参照して
スナップショット複写する (ADR-[[202606190900]])。クロスドメインの直接書き込みはコード上存在しない。

しかし**強制が規約レベルに留まっていた**。全サービスが共有スーパーユーザ `ec` で DB に接続して
いたため、同居 schema (db-customer の `order` / `payment` / `member`、db-ops の `product` /
`shipping`) へは技術的には誰でも書けた。`order` の SQL に `payment.payments` への INSERT を
書いても DB は止めない。ADR-[[202606170909]] は「権限分離 (Postgres role 等) は Issue で別途
扱う」とこの穴を明示的に先送りしていた。

「所有権を確定」を計測可能な不変条件にするには、DB の ACL で「自 schema 以外は触れない」を
強制する必要がある。

## Decision

**サービスごとに最小権限のログインロールを 1:1 で用意し、自 schema にしか DML できないよう
GRANT で強制する。** DDL とロールへの付与は引き続き管理ユーザ `ec` が行う。

- ロール: `order_svc` / `payment_svc` / `member_svc` (db-customer)、`product_svc` /
  `shipping_svc` (db-ops)。各ロールは自 schema に `USAGE` + テーブル
  `SELECT/INSERT/UPDATE/DELETE` + シーケンス `USAGE,SELECT` のみを持つ。他 schema への
  `USAGE` を与えないため、PostgreSQL 17 では他 schema のオブジェクトは参照すらできない。
- **DDL は `ec` が所有・実行**する (migration は従来通り)。サービスロールに DDL 権限は与えない
  ので、runtime での schema 変更も不可能になる ＝ 所有の一部。
- ロール生成はクラスタ全体のオブジェクトで schema migration の単位 (DB 内) に収まらないため、
  DB 初期化レイヤ (`infra/db/{customer,ops}/initdb/01-roles.sql` を `docker-entrypoint-initdb.d`
  にマウント) で行う。
- GRANT は各サービスの migration (`server/<svc>/db/migration/*_grant_service_role.sql`) に置き、
  `ec` が自ロールへ付与する。schema 単位の権限ポリシーを所有サービスが持つ
  (ADR-[[202606180900]] のドメインオーナーシップに沿う)。将来 `ec` が追加する table/sequence にも
  効くよう `ALTER DEFAULT PRIVILEGES FOR ROLE ec` を併用する。
- サービスは runtime で自ロールの DSN で接続する (`compose.yaml`)。SQL は全て schema 修飾済みの
  ため search_path 調整は不要。

## Consequences

- **所有権が DB ACL の不変条件になる**。`order_svc` で `SELECT * FROM payment.payments` は
  `permission denied for schema payment` になり、「他ドメインへの書き込みは持ち主サービス経由」が
  規約でなく強制になる。これは psql で計測して確認できる (推測でなく計測)。
- **migration / 結合テストの仕組みは無変更**。migrate は `ec`、テストの clone
  (`CREATE DATABASE ... TEMPLATE`、ADR-[[202606190902]]) も `ec` のまま。最小権限の実証は
  サービスが自ロールで動く E2E が担う。
- GRANT migration はロール存在をガードするため、initdb を通らない経路 (testcontainers 単体等) でも
  no-op で通る。
- initdb はデータ volume が空の初回のみ実行される。ロール定義を変えた・既存 volume へ反映したい
  ときは `docker compose down -v` が要る (学習用ローカル前提として許容)。
- ロールのパスワードは local/study 用に `<svc>_pass` 固定。本番想定の secret 管理 (Vault・IAM 認証
  等) は本 ADR の範囲外。
- **Step 3 への布石**: DB をドメイン単位に物理分割する際、各 `*_svc` ロールがそのまま分割後 DB の
  所有者に育てられる。

## Alternatives considered

- **各ロールが自 schema を所有し migration も自前実行**: 所有権はより純粋になるが、`migrate.sh` /
  goose version table / 結合テストの `ec` 前提を作り変える必要があり churn が大きい。学習用の段階で
  得られる強制力は GRANT 方式と同じため不採用。
- **search_path をロールごとに固定するだけ**: 既定 schema を狭めても `payment.payments` のように
  明示修飾すれば到達できてしまい、強制にならない。
- **強制せず ADR / README に所有権を明文化するだけ**: 「order が payment に書けてしまう」穴がコード上
  残る。Step 2 の主目的 (所有権の確定) を満たさない。
- **アプリ層で接続先 schema を検査するミドルウェア**: DB の外で番人を立てる二重管理。DB 自身の
  ACL で表現できるものをアプリに持ち込むのは本末転倒。
