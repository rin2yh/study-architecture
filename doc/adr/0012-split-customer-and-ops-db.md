# ADR 0012: 顧客系 (社外) と運用系 (社内) で DB とネットワーク経路を分離

- Status: Proposed
- Date: 2026-06-17
- Supersedes (一部): [[0004]] の「共有 Postgres 1 つ」前提

## Context

[[0004]] では Step 0 のシンプルさを優先して **1 つの Postgres インスタンスをドメイン
schema で区画**していた。[[0001]] のロードマップでも Step 3 で DB 分割は予定して
いるが、**分割軸は「ドメイン」(payment / shipping → member → product × order)** で、
「**顧客向け / 裏側 (運用)**」という軸は明示されていなかった。

EC の現場では「顧客系 (受発注・決済・会員) は OLTP 要件が厳しく可用性最優先、運用系
(商品マスタ・在庫・配送) は集計や bulk 操作が混じり SLA 要件が異なる」のが定石で、
**両者を同じインスタンスに同居させると運用クエリが顧客 SLA を傷つけうる**。早い段階で
物理分離して「混ぜない」ことを既定にしておきたい。

## Decision

**顧客系 (db-customer)** と **運用系 (db-ops)** で **Postgres インスタンスを物理分離**
する。Step 0 の段階で 2 インスタンス構成にする。

- db-customer (顧客操作の write が主): `order`, `payment`, `member`
- db-ops (運用操作の write が主): `product`, `shipping`

各サービスは **自身の所有 DB だけを向く**。横断は禁止し、横断データは [[0008]] の
スナップショット保存で吸収する。

### Implementation

- `compose.yaml` に `db-customer` (ホスト 5432) と `db-ops` (ホスト 5433) の 2 サービスを
  立てる。コンテナ内では両方とも 5432 を listen する。
- `db/migration/` を 2 つのディレクトリに分割する。
  - `db/migration/customer/` … order / payment / member の schema と table
  - `db/migration/ops/` … product / shipping の schema と table
- 各サービスの `sqlc.yaml` の `schema:` を自身が所属する側 (`db/migration/customer` か
  `db/migration/ops`) に向け、生成型は所有 DB の schema にスコープする。
- 各サービスの `DATABASE_URL` を compose で当該 DB に向ける。
- goose は **ディレクトリ単位** で migration を管理する設計なので、`mise migrate` を
  `migrate:customer` / `migrate:ops` に分けて両方適用する。
- compose の `migrate` profile も 2 系統 (`migrate-customer`, `migrate-ops`) に分ける。

### Network topology (Reverse Proxy パターン)

DB 分離だけでなく **Docker network レベルでも経路を分け** て「社外 → 社内」アクセスを
落とす。実世界のクラウド構成 (Public subnet / Private subnet + API Gateway / Load Balancer)
を Docker compose に落とすと **Reverse Proxy パターン**になる。業務サービスが両 network に
attach する (= マルチホームホスト) のは不自然なので、**両方に足を持つのは reverse proxy 1
個だけ**に限定する。

- **external network (社外公開)**: 顧客 UI (store / mypage)、edge-proxy
- **internal network (社内専用)**: 運用 UI (backoffice)、全 backend サービス、
  db-customer、db-ops、edge-proxy

各サービスの network 帰属:

| サービス | external | internal |
| --- | :---: | :---: |
| store / mypage (顧客 UI) | ✓ | - |
| backoffice (運用 UI) | - | ✓ |
| edge-proxy (nginx) | ✓ | ✓ |
| product / order / payment / member / shipping | - | ✓ |
| db-customer / db-ops | - | ✓ |

経路:

- **顧客 UI → backend**: store / mypage は `http://edge-proxy/<svc>/...` を呼び、
  edge-proxy が `location /<svc>/` で内部の `<svc>:8080` に proxy する。
- **運用 UI → backend**: backoffice は internal 内で `http://<svc>:8080` を直接叩く。
- **社外 → 社内**: edge-proxy が許可した path 以外は届かない (例: db-ops は名前解決不可)。
- **社内 → 社外**: 運用 UI から社外 UI への直接アクセス経路は無い (要件)。

帰結:

- 「両方の network に attach する container」を業務サービスではなく proxy だけに局所化でき、
  実世界の DMZ 構成 (リバプロが両ゾーンに足を持つ) と表現が一致する。
- 顧客 UI 側は backend のサービス名を知らず、edge-proxy のホスト名と path だけ知っていれば
  よい。将来 Step 1 で BFF を入れるときに edge-proxy が BFF/API Gateway へ自然に育つ。
- 権限分離 (Postgres role 等) は Issue で別途扱う。

## Consequences

- **可用性**: 運用バッチ・集計が顧客 OLTP を巻き込まない。顧客系のメンテナンスウィンドウを
  独立して取れる。
- **権限**: 顧客系と運用系で DB ユーザを分離でき、最小権限の運用がしやすくなる。
- **横断 JOIN は禁止**: order が product を JOIN するような問い合わせは書けない。
  必要な属性は注文確定時に order 側に **スナップショット保存** ([[0008]]) する。
- **マイグレーション管理**: ディレクトリ 2 つに分け、goose の version table も独立する。
  新規 schema 追加時に「どちらに置くか」を ADR/PR で必ず明示する運用が必要。
- **テスト**: testcontainers で統合テストを書く際は、各サービスの DB を独立に立てる前提に
  なる (#3 の Issue で対応)。
- **ロードマップへの影響**: [[0001]] の Step 3 で予定していた「ドメイン軸の分割」は依然
  目標だが、Step 0 でまず「顧客 / 運用」の粗い軸を入れておく。Step 3 では各 instance を
  さらにドメイン単位の cluster に分ける余地を残す。

## Alternatives considered

- **read replica で読み取り分離**: 集計負荷の軽減には効くが、運用 write (商品マスタ更新等)
  と顧客 write の競合は解消しない。本 ADR の目的に合致しない。
- **ADR 0004 のまま Step 3 まで遅延**: ロードマップどおり「最小で動かす」立場には素直だが、
  EC の定石としての「顧客 / 運用分離」を後付けすると schema やデータ移行コストが大きい。
  Step 0 のうちに物理分離しておく方が後の修正コストが小さい。
- **API レベルで読み取り経路を分ける**: backoffice 用 API を read-only replica に向ける手は
  あるが、書き込み混在を防げない。本 ADR は「インスタンス分離 = 書き込みの境界」を採る。
