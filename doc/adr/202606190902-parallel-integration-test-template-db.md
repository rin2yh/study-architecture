# ADR-202606190902: 結合テストはテンプレート DB クローンで分離し並列実行する

- Status: Accepted
- Date: 2026-06-19
- Related: ADR-[[202606170909]] (customer/ops 物理 DB 分割) / ADR-[[202606180900]] (migration をサービスごとに分割) / ADR-[[202606180902]] (repository は実 DB 結合テストで検証)

## Context

ADR-[[202606180902]] で repository / handler を実 DB の結合テストで検証するようにした。接続先は
`DATABASE_URL_CUSTOMER` / `DATABASE_URL_OPS` が指す**共有 DB** (`ec_customer` /
`ec_ops`) で、各テストは対象 table を `TRUNCATE ... RESTART IDENTITY` してから seed する。

この構成では同じ物理 DB の同じ table を複数テストが奪い合う。Go はパッケージを並列
(`-p`) 実行するため、`member/internal/repository` と `member/internal/handler` を同時に
走らせると、一方の TRUNCATE が他方の seed を消すなどの競合が起きる。そのため CI の
`server-integration` job と `mise run test:integration` は **`-p 1` (逐次)** で逃げていた。

逐次実行は安全だが、結合テストが増えるほど直線的に遅くなる。サービス数 5 × (repository +
handler) でパッケージが増えており、並列化したい。

## Decision

**テストごとに DB を分離する。`testdb.Open` を「DSN が指す DB をテンプレートに
`CREATE DATABASE ... TEMPLATE` でクローンし、そのクローンへの接続を返す」ヘルパーに変える。**

- `Open` 呼び出し 1 回 = クローン DB 1 個。テスト関数ごとに独立した DB を持つので、
  TRUNCATE/seed が他テストと衝突しない。
- クローン名は `<template>_t_<8byte hex>`。プロセスをまたいだ並列実行でも衝突しないよう
  `crypto/rand` で採番する (`go test` はパッケージごとに別プロセス)。
- `CREATE`/`DROP DATABASE` は対象 DB 接続中・トランザクション内では実行できないため、
  維持用 DB (`postgres`) への単発接続から発行する。utility 文は extended protocol で
  prepare できないので simple protocol を既定にした接続を使う。
- 後始末は `t.Cleanup` で `pool.Close()` → `DROP DATABASE ... WITH (FORCE)` の順
  (登録は LIFO なので DROP を先に登録)。FORCE は残接続があっても落とすため (PG13+)。

**これにより `-p 1` を撤去する。**

- CI `server-integration` job と `mise run test:integration` の `go test` から `-p 1` を外す。
- 既存テスト本体 (TRUNCATE+seed) は無改修。変更は `server/internal/test/db` ヘルパーに閉じる。
- 今回は各テストへの `t.Parallel()` 付与は**行わない**。まずパッケージ間並列 (`-p` 既定) で
  効果を取り、パッケージ内並列は分離が効くと確認できてから別途入れられる (Open がテスト
  単位で分離するので後付けは安全)。

テンプレートは ADR-[[202606180900]] の `scripts/migrate.sh` が migration を流した `ec_customer` /
`ec_ops` をそのまま使う。クローンは migration 済み状態を丸ごと引き継ぐので、テスト側で
流し直す必要はない。

## Consequences

- **並列化で結合テストが短縮される**: `-p 1` を外し、パッケージを並列実行できる。
- **分離が強い**: テストごとに別 DB なので、TRUNCATE/seed の競合だけでなく
  「前のテストが残したデータ」による相互汚染も原理的に起きない。
- **テンプレートに接続してはいけない**: `CREATE DATABASE ... TEMPLATE foo` は `foo` に
  アクティブ接続があると失敗する。テストはクローンにのみ接続し、テンプレート
  (`ec_customer` / `ec_ops`) には直接つながない前提を守る。
- **superuser/CREATEDB 権限が要る**: クローン作成にはその権限が必要。compose の `ec`
  ユーザ (postgres 初期化ユーザ = superuser) で満たす。別環境で回すときは権限に注意。
- **クローンの作成/破棄コストが乗る**: テンプレートは数 table と小さく、`CREATE DATABASE`
  はファイルコピーなので軽い。migration を毎回流す方式より速い。
- **CI のセットアップは不変**: DB 起動 → migrate の流れ (ADR-[[202606180900]]/ADR-[[202606180902]]) はそのまま。
  migrate がテンプレートを用意し、テストがそこからクローンする。

## Alternatives considered

- **`-p 1` のまま据え置き**: 最も安全だが、結合テストが増えるほど遅くなる。並列化要件に
  反するので退けた。
- **トランザクションロールバックで分離 (テストごとに BEGIN→ROLLBACK)**: DB を増やさず速い
  が、`pgxpool` 越しの複数接続・`TRUNCATE`・複数文をまたぐ検証と相性が悪く、既存テスト
  (pool を共有し複数クエリを投げる) を大きく書き換える必要がある。Open に閉じない。
- **schema をテストごとに切り替える (search_path)**: 物理 DB を増やさず分離できるが、
  sqlc 生成クエリが schema 修飾 (`member.members` 等) を埋め込んでおり search_path で
  逃がせない。テンプレート DB クローンの方が既存コードに手を入れずに済む。
- **testcontainers でテストごとに Postgres を起動**: 分離は最も強いがコンテナ起動が重く、
  ADR-[[202606180902]] で既存 compose DB を流用すると決めた方針とも整合しない。テンプレートクローンは
  同一インスタンス内の安価な複製で目的を満たす。
- **パッケージごとに DB を分ける (TestMain で 1 DB)**: クローン数は減るが各 integration
  パッケージに `TestMain` を足す必要があり、パッケージ内のテーブル共有 (TRUNCATE 競合) は
  残る。Open 呼び出しごとの分離なら変更がヘルパーに閉じ、分離も強い方を採った。
