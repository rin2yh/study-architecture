# ADR-202606231000: データ所有権を schema ごとの最小権限 DB ロールで強制する

- Status: Accepted
- Date: 2026-06-23
- Relates to: ADR-[[202606170903]] / ADR-[[202606170909]] / ADR-[[202606180900]] / ADR-[[202606190900]]

## Context

- ロードマップ Step 2「データ所有権を確定し schema 分離を徹底」(ADR-[[202606170900]]) は、アプリ層では既に満たされている (各サービスは自 schema しか触らず、横断は API 参照のスナップショット複写 ADR-[[202606190900]])。
- だが強制が規約レベルに留まっていた。全サービスが共有スーパーユーザ `ec` で接続するため、同居 schema へは技術的に誰でも書け、`order` の SQL から `payment.payments` へ INSERT しても DB は止めない。権限分離は ADR-[[202606170909]] で先送りされていた。
- 「所有権の確定」を計測可能な不変条件にするには、DB の ACL で「自 schema 以外は触れない」を強制する必要がある。

## Decision

**サービスごとに最小権限のログインロールを 1:1 で用意し、自 schema にしか DML できないよう GRANT で強制する。** DDL とロール付与は引き続き管理ユーザ `ec` が行う。

- 各 `*_svc` ロールは自 schema にのみ DML 権限を持つ。他 schema への `USAGE` を与えないため、PostgreSQL 17 では他 schema のオブジェクトは参照すらできない。
- DDL とサービスロールへの GRANT は `ec` が所有・実行する。goose migration は無変更。サービスロールに DDL を与えないので runtime での schema 変更も不可になる (所有の一部)。
- **ロール生成と GRANT は goose の外で `ec` が冪等に適用する。** 決め手: GRANT は「一方向 migration」ではなく「再適用する宣言的ポリシー」なので、version 追跡される goose には載せず何度でも流せる形にする。`ec` が将来追加する table/sequence にも効くよう `ALTER DEFAULT PRIVILEGES` を併用。
- 適用 psql は postgres コンテナ内 (`docker compose exec`) を使う。決め手: host への psql 依存を避け (CI の unit/integration は psql 未保証)、権限適用をスタック起動済みの E2E / ローカル経路に限定するため。
- サービスは runtime で自ロールの DSN で接続する。SQL は全て schema 修飾済みのため search_path 調整は不要。

## Consequences

- 所有権が DB ACL の不変条件になる。`order_svc` での `SELECT * FROM payment.payments` は `permission denied` になり、規約でなく強制になる (psql で計測確認できる)。
- migration / 結合テストの仕組みは無変更。migrate もテスト clone (ADR-[[202606190902]]) も `ec` のまま。最小権限の実証はサービスが自ロールで動く E2E が担う。
- 権限ポリシーが 1 スクリプトに集約され再実行可能。ロール変更・新サービス追加は流し直すだけで `down -v` を要さず、冪等なので二重適用も安全。
- ロールのパスワードは local/study 用の固定値。本番想定の secret 管理 (Vault・IAM 認証等) は範囲外。
- Step 3 への布石: DB をドメイン単位に物理分割する際、各 `*_svc` ロールを分割後 DB の所有者に育てられる。

## Alternatives considered

- **GRANT を各 goose migration に、ロール生成を initdb に置く** (本 ADR の初稿): 権限が 7 ファイルに分散し、goose 管理の有無が migration と initdb で非対称になる。整合 (ロールが先に在る前提) が暗黙でガードが要り、ロール変更時は initdb が再走らず `down -v` 必須で壊れやすい。GRANT は宣言的ポリシーなので再実行可能な 1 スクリプトに集約する本決定へ差し替えた。
- **各ロールが自 schema を所有し migration も自前実行**: 所有はより純粋だが `migrate.sh` / goose version table / 結合テストの `ec` 前提を作り変える churn が大きく、学習段階で得られる強制力は GRANT 方式と同じため不採用。
- **search_path をロールごとに固定するだけ**: 明示修飾 (`payment.payments`) で到達でき強制にならない。
- **強制せず ADR / README に明文化するだけ**: 「order が payment に書ける」穴がコードに残り、Step 2 の主目的を満たさない。
- **アプリ層で接続先 schema を検査するミドルウェア**: DB の外に番人を立てる二重管理。ACL で表現できるものをアプリに持ち込むのは本末転倒。
