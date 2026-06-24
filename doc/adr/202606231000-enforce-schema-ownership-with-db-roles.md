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
- **DDL (schema / table) は `ec` が所有・実行**する。goose migration は従来通り (無変更)。
  サービスロールに DDL 権限は与えないので、runtime での schema 変更も不可能になる ＝ 所有の一部。
- **ロール生成と GRANT は goose の外で `ec` が冪等に適用する**。GRANT は「スキーマ進化
  (一方向 migration)」ではなく「再適用する宣言的ポリシー」なので、version 追跡される goose ではなく、
  何度でも流せる形で表現する。SQL の実体は `scripts/grant.{customer,ops}.sql` に置き、`scripts/grant.sh`
  はそれを各 DB へ流すだけにする (heredoc 埋め込みを避け、SQL を `.sql` として持つ)。将来 `ec` が
  追加する table/sequence にも効くよう `ALTER DEFAULT PRIVILEGES FOR ROLE ec` を併用する。
- `grant.sh` は psql を **postgres コンテナ内のもの** (`docker compose exec`) で使う。これにより
  `migrate.sh` は host への psql 依存を持たずに済み (CI の unit/integration は psql 未保証)、権限適用は
  スタックが起動している E2E / ローカル経路に限定できる。`scripts/e2e-up.sh` が migrate 直後に呼ぶ。
- サービスは runtime で自ロールの DSN で接続する (`compose.yaml`)。SQL は全て schema 修飾済みの
  ため search_path 調整は不要。

## Consequences

- **所有権が DB ACL の不変条件になる**。`order_svc` で `SELECT * FROM payment.payments` は
  `permission denied for schema payment` になり、「他ドメインへの書き込みは持ち主サービス経由」が
  規約でなく強制になる。これは psql で計測して確認できる (推測でなく計測)。
- **migration / 結合テストの仕組みは無変更**。migrate は `ec`、テストの clone
  (`CREATE DATABASE ... TEMPLATE`、ADR-[[202606190902]]) も `ec` のまま。CI の unit / server-integration
  は `ec` で動くので grant 自体が不要。最小権限の実証はサービスが自ロールで動く E2E が担う。
- **権限ポリシーが 1 ファイルに集約され、再実行可能**。ロール定義変更・新サービス追加は `grant.sh`
  を流し直すだけで、`docker compose down -v` を要しない。冪等なので二重適用も安全。
- ロールのパスワードは local/study 用に `<svc>_pass` 固定。本番想定の secret 管理 (Vault・IAM 認証
  等) は本 ADR の範囲外。
- **Step 3 への布石**: DB をドメイン単位に物理分割する際、各 `*_svc` ロールがそのまま分割後 DB の
  所有者に育てられる。

## Alternatives considered

- **GRANT を各サービスの goose migration に、ロール生成を initdb に置く** (本 ADR の初稿): 権限が
  7 ファイル (initdb 2 + migration 5) に分散し、「version 追跡される migration」と「初回ブート専用の
  initdb」で goose 管理の有無が非対称になる。両者の整合 (ロールが先に在る前提) は暗黙で、migration 側に
  ロール存在ガードが要る。ロール定義変更時は initdb が再走らず `down -v` 必須で壊れやすい。GRANT は
  宣言的ポリシーなので、再実行可能な 1 スクリプトに集約する本決定に差し替えた。
- **各ロールが自 schema を所有し migration も自前実行**: 所有権はより純粋になるが、`migrate.sh` /
  goose version table / 結合テストの `ec` 前提を作り変える必要があり churn が大きい。学習用の段階で
  得られる強制力は GRANT 方式と同じため不採用。
- **search_path をロールごとに固定するだけ**: 既定 schema を狭めても `payment.payments` のように
  明示修飾すれば到達できてしまい、強制にならない。
- **強制せず ADR / README に所有権を明文化するだけ**: 「order が payment に書けてしまう」穴がコード上
  残る。Step 2 の主目的 (所有権の確定) を満たさない。
- **アプリ層で接続先 schema を検査するミドルウェア**: DB の外で番人を立てる二重管理。DB 自身の
  ACL で表現できるものをアプリに持ち込むのは本末転倒。
