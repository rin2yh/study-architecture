# ADR-202606190902: 結合テストはテンプレート DB クローンで分離し並列実行する

- Status: Accepted
- Date: 2026-06-19
- Related: ADR-[[202606170909]] (customer/ops 物理 DB 分割) / ADR-[[202606180900]] (migration をサービスごとに分割) / ADR-[[202606180902]] (repository は実 DB 結合テストで検証)

## Context

ADR-[[202606180902]] の結合テストは**共有 DB** に接続し、各テストが対象 table を TRUNCATE+seed する。同じ物理 DB の table を複数テストが奪い合うため、Go のパッケージ並列 (`-p`) で TRUNCATE と seed が競合する。CI と `mise run test:integration` は `-p 1` (逐次) で逃げていたが、結合テストが増えるほど直線的に遅くなり並列化できない。

## Decision

**テスト単位で DB を分離する。`testdb.Open` を「DSN の DB を `CREATE DATABASE ... TEMPLATE` でクローンし、そのクローンへの接続を返す」ヘルパーに変え、`-p 1` を撤去する。**

- 決め手: `Open` 1 回 = 独立した DB 1 個なので、TRUNCATE/seed が他テストと衝突しない。変更は `server/internal/test/db` ヘルパーに閉じ、既存テスト本体は無改修。
- クローン名は `crypto/rand` で採番する。`go test` がパッケージごとに別プロセスで走るため、プロセスをまたいでも衝突させないため。
- テンプレートは ADR-[[202606180900]] の migration 済み `ec_customer` / `ec_ops` を流用する。クローンが migration 済み状態を引き継ぐので流し直し不要。
- `t.Parallel()` 付与は今回やらない。まずパッケージ間並列で効果を取り、Open がテスト単位で分離するので後付けは安全。

## Consequences

- **結合テストが短縮される**: `-p 1` を外しパッケージを並列実行できる。
- **分離が強い**: テストごとに別 DB なので、競合だけでなく前テストの残データによる相互汚染も原理的に起きない。
- **クローン作成/破棄コストが乗る**: テンプレートは小さく `CREATE DATABASE` はファイルコピーで軽い。migration を毎回流す方式より速い。
- **制約**: テンプレート (`ec_customer` / `ec_ops`) にはアクティブ接続があるとクローンが失敗するため直接つながない。クローン作成に superuser/CREATEDB 権限が要る (compose の `ec` ユーザで満たす。別環境では要注意)。
- **CI のセットアップは不変**: DB 起動 → migrate (ADR-[[202606180900]]/ADR-[[202606180902]]) はそのまま。migrate がテンプレートを用意する。

## Alternatives considered

- **`-p 1` 据え置き**: 最も安全だが結合テストが増えるほど遅く、並列化要件に反する。
- **トランザクションロールバックで分離**: 速いが、pool 共有で複数クエリ・TRUNCATE を投げる既存テストと相性が悪く、大幅な書き換えが要り Open に閉じない。
- **schema をテストごとに切り替える (search_path)**: sqlc 生成クエリが schema 修飾を埋め込んでおり search_path で逃がせない。
- **testcontainers でテストごとに Postgres 起動**: 分離は最強だが起動が重く、既存 compose DB を流用する ADR-[[202606180902]] の方針と整合しない。
- **パッケージごとに DB を分ける (TestMain で 1 DB)**: クローン数は減るが各パッケージに TestMain が要り、パッケージ内の TRUNCATE 競合が残る。Open 単位の分離の方が変更が閉じ分離も強い。
