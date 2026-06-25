# ADR-202606170909: 顧客系 (社外) と運用系 (社内) で DB とネットワーク経路を分離

- Status: Accepted (Step 3 で各 instance をドメイン単位へ分割完了。ADR-[[202606240522]])
- Date: 2026-06-17
- Supersedes (一部): ADR-[[202606170903]] の「共有 Postgres 1 つ」前提

## Context

ADR-[[202606170903]] は Step 0 のシンプルさを優先して 1 つの Postgres をドメイン schema で区画していた。ADR-[[202606170900]] のロードマップでも DB 分割は Step 3 の予定だが、分割軸は「ドメイン」で、「顧客向け / 裏側 (運用)」という軸は無かった。EC では顧客系 (受発注・決済・会員) は OLTP で可用性最優先、運用系 (商品マスタ・在庫・配送) は集計・bulk が混じり SLA 要件が異なり、同居させると運用クエリが顧客 SLA を傷つけうる。早い段階で物理分離して「混ぜない」を既定にしたい。

## Decision

**顧客系 (db-customer: order/payment/member)** と **運用系 (db-ops: product/shipping)** で Postgres インスタンスを物理分離し、Step 0 から 2 インスタンス構成にする。各サービスは自身の所有 DB だけを向き、横断は禁止、横断データは ADR-[[202606190900]] のスナップショットで吸収する。

- migration を `db/migration/{customer,ops}/` に分け、各サービスの `sqlc.yaml` / `DATABASE_URL` を所属側に向ける。goose はディレクトリ単位管理なので適用も 2 系統に分ける。
- 同居サービス群の on/off は Docker compose の `profiles:` (`external` / `internal`) で切り替える。compose 公式の正攻法のため、公開ゾーンでファイルを割る案は採らない。

### Network topology (4 subnet)

DB だけでなく Docker network でも経路を分け「社外 → 社内」アクセスを落とす。クラウドの VPC + public/private subnet を模し、**社外側・社内側それぞれに public/private を持つ 4 network 構成** (`external-public` / `external-private` / `internal-public` / `internal-private`) にする。

両方の network に足を持つのは **2 種の proxy (edge-proxy / backoffice) のみ**に局所化し、業務サービスは multi-home させない。これが実世界の「DMZ proxy」「社内 admin gateway」と表現一致する。帰結する到達性:

- 顧客 UI → 顧客系/運用系 backend は edge-proxy が中継 (DMZ proxy が許可した path のみ)。
- 運用 UI (backoffice) は internal/external-private 双方に attach し、運用系・顧客系 backend へ直接到達 (「社内 → 社外 OK」)。
- 顧客 UI は `external-public` にしか居らず internal-* を名前解決できないため社内へ到達不能 (「社外 → 社内 NG」)。

顧客 UI は backend のサービス名を知らず edge-proxy のホスト名と path だけ知ればよく、将来 Step 1 の BFF / API Gateway へ edge-proxy が自然に育つ。

## Consequences

- **可用性**: 運用バッチ・集計が顧客 OLTP を巻き込まない。顧客系のメンテ枠を独立に取れる。
- **権限**: 顧客系/運用系で DB ユーザを分離でき最小権限化しやすい (Postgres role の強制は別 Issue)。
- **横断 JOIN 禁止**: 必要な属性は注文確定時に order 側へスナップショット (ADR-[[202606190900]])。
- **マイグレーション**: ディレクトリ 2 つで goose の version table も独立。新規 schema は「どちらに置くか」を PR で明示する運用が要る。
- **テスト**: 統合テストは各サービスの DB を独立に立てる前提になる。
- **ロードマップ**: Step 3 のドメイン軸分割は依然目標で、各 instance をさらにドメイン単位 cluster へ分ける余地を残す。

## Alternatives considered

- **read replica で読み取り分離**: 集計負荷は軽くなるが運用 write と顧客 write の競合は解消しない。
- **ADR-[[202606170903]] のまま Step 3 まで遅延**: 「最小で動かす」には素直だが、顧客/運用分離を後付けすると schema・データ移行コストが大きい。
- **API レベルで読み取り経路を分ける**: 書き込み混在を防げない。本 ADR は「インスタンス分離 = 書き込みの境界」を採る。
