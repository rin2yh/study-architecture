# ADR-202606240522: Step 3 で結合の弱い縁からドメインごとに DB インスタンスを分割する

- Status: Accepted
- Date: 2026-06-24
- Relates to: ADR-[[202606170909]] (顧客系/運用系の粗い DB 分離), ADR-[[202606190900]] (横断スナップショット), ADR-[[202606231000]] (schema 所有権の最小権限ロール), ADR-[[202606180900]] (migration 分割)

## Context

ADR-[[202606170909]] は Step 0 の段階で DB を **顧客系 (db-customer: order/payment/member)** と
**運用系 (db-ops: product/shipping)** の 2 インスタンスへ粗く分けた。同 ADR の Consequences は
「Step 3 で各 instance をさらにドメイン単位へ分ける余地を残す」と明記している。

issue #9 (Step 3) のゴールはこの細分化で、分割順序は **payment / shipping → member →
product × order**。これは「結合の弱い縁から剥がす」並びで、最も中心的で業務影響の大きい
order × product を最後に回す。横断 JOIN は ADR-[[202606190900]] のスナップショットで既に
無くなっているため、DB を物理分割しても照会は各サービスの自 DB で閉じる。

## Decision

Step 3 では **ドメインごとに独立した Postgres インスタンス**へ、結合の弱い縁から順に剥がす。
issue は #9 を親に Step 3-1〜3-4 (payment / shipping / member / product×order) のサブイシューへ分け、
**1 PR で 1 ドメインずつ**進める。

### 分割順序を「弱い縁から」にする理由

依存の少ない縁ほどデータ移行・経路変更の影響範囲が小さく、失敗時に巻き戻しやすい。先に
簡単な所で分割手順 (compose / migrate / grant / test / CI の同型変更) を確立し、知見を貯めてから
中心の order × product に着手する。漸進的なリスク管理であって、トラフィック量やアルファベット順は
判断軸ではない。

### Step 3-1: payment を db-payment へ (本 ADR で着手)

- `compose.yaml` に `db-payment` (host 5434 / `ec_payment`, external-private) を追加し、
  payment の `DATABASE_URL` を向け替える。order / member は db-customer に残す。
- migration は ADR-[[202606180900]] どおりサービス単位。payment の `00001_init_schema.sql` が
  `CREATE SCHEMA IF NOT EXISTS payment` を持つので、空の `ec_payment` へ流せば schema ごと作られる。
- 最小権限ロール (ADR-[[202606231000]]) は維持し、`payment_svc` の作成・GRANT を
  `scripts/grant/customer.sql` から `scripts/grant/payment.sql` へ分離する。
- 結合テストの DSN env を `DATABASE_URL_CUSTOMER` → `DATABASE_URL_PAYMENT` に向け替える。
  テンプレート DB クローン方式 (ADR-[[202606190902]]) は対象インスタンスが変わるだけで不変。

後続 (shipping / member / product×order) も同型の変更で進める。

## Consequences

- **可用性の独立**: payment のメンテナンス・障害が order/member の DB を巻き込まなくなる。
  決済系を独立してスケール・バックアップできる。
- **インスタンス増**: 立てる Postgres と接続経路・監視対象が増える。compose / CI / mise の
  起動対象に db-payment を足す手間が各 Step で発生する。
- **横断は従来どおり禁止**: order→payment は同期 HTTP、order→product はスナップショットで、
  DB 横断 JOIN は無いまま。分割で新たな結合は生まない。
- **テスト分離**: payment の結合テストは専用 DSN を要求する。CI の server-integration は
  db-customer と db-payment の双方を起動して migrate する。
- **段階移行**: issue #9 は 1 PR 1 ドメインで進むため、db-customer/db-ops は分割の途中状態を
  経る (例: payment 剥離後の db-customer は order/member の 2 schema)。最終形は 3-4 で確定する。

## Alternatives considered

- **issue #9 を 1 PR で 5 ドメイン一括分割**: 変更が大きくレビュー困難で、弱い縁で手順を
  確立する学習効果も失う。本 ADR は 1 ドメインずつに割る。
- **customer/ops の 2 インスタンスで据え置く**: ADR-[[202606170909]] の段階としては妥当だが、
  Step 3 のゴール (ドメイン軸の所有権確定) に未達。
- **中心 (order×product) から分割**: 影響の大きい所を最初に触ることになり、手順未確立の
  まま最高リスクを引く。弱い縁からの逆。
