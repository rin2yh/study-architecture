# ADR-202606190903: repository を CQRS で Query / Command に分割する

- Status: Accepted
- Date: 2026-06-19
- Related: ADR-[[202606180902]] (repository は実 DB 結合テストで検証) / ADR-[[202606180903]] (更新エンドポイントの PUT セマンティクス)

## Context

handler は既に読み (`xxx_read.go`) / 書き (`xxx_write.go`) で分かれているのに、依存する repository だけが読み書きを 1 つの `XxxRepository` interface + `Repository` struct に束ねていた。この非対称により:

- 読みハンドラが書き込みメソッドごと依存し、「何を触るか」が型から読めない。
- 将来 read 側だけ別経路 (キャッシュ / read replica) に差し替えたくても、書き込みと同じ型に縛られる。

## Decision

**repository を CQRS の語彙で Query (読み) / Command (書き) に分割する。**

- 具象 struct を `XxxQuery` / `XxxCommand` に分ける (sqlc 生成の `db.Querier` を保持する薄い委譲層なのは従来どおり)。
- パッケージ / ディレクトリ名は `repository` → `rdb`。CQRS 分割後は単一の「リポジトリ」集約が無く、`rdb` は「RDB アクセスの具象実装が集まる場所」を表す。
- 抽象 (interface) は利用側の handler パッケージに `Query` / `Command` として置く (consumer interface)。Go の「受け取りは interface、返すは具象」に従い `rdb` から interface を撤去。
- handler も型を分け、`readHandler` は `Query` のみ・`writeHandler` は `Command` のみを持つ。これにより「読みから書き込みを呼べない」ことを型で保証する。公開 `Handler` は両者を埋め込んで単一の `api.ServerInterface` を満たす (codegen のルータが 1 ServerInterface を要求するため合成型が要る)。
- DI (kessoku) の `Bind` は具象→interface の暗黙変換をしないため、consumer interface だが `Query` / `Command` は export する。
- stub は据え置き。構造的部分型で `handler.Query` / `handler.Command` 両方を満たすので、読み/書きどちらのテストでも同じ stub を使える。

## Consequences

- **読み書きの依存が型で分離**: handler の責務分割 (ADR-[[202606180902]]) と repository の形が揃う。
- **read 側を独立して差し替える拡張点ができた**: 現状は単一 pool を共有 (read replica の必要性は未計測)。
- **テストの注入点は維持**: `rdb` は実 DB 結合テスト、handler は stub 分岐網羅という二段構え (ADR-[[202606180902]]) はそのまま。
- **注意点**: consumer interface の export は Go idiom からやや外れるが、kessoku の `Bind` が export 済み interface を要求するための割り切り。

## Alternatives considered

- **interface だけ 2 分割し具象 struct は 1 つのまま**: 最小変更だが、1 struct が読み書き両方を実装し続け「read 側を差し替える」拡張点が生まれず、CQRS の動機を半分しか満たさない。
- **interface を具象と同じパッケージに置く (`rdb.MemberQuery` 等)**: 具象を `MemberQuery` と名付けたい要件と名前が衝突する。consumer interface を handler 側に置くことでリソース名を冠した具象名 (DI / テストで可読性が高い) を優先した。
- **read / write で pool を分割する**: 将来の read replica 向け物理分割。別 DB に分ける必要性は未計測で、ADR-[[202606170909]] の customer/ops 分割で十分。型分離だけ先に入れ、pool 分割は必要になってから行う。
