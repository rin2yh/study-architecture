# ADR-202606190903: repository を CQRS で Query / Command に分割する

- Status: Accepted
- Date: 2026-06-19
- Related: ADR-[[202606180902]] (repository は実 DB 結合テストで検証) / ADR-[[202606180903]] (更新エンドポイントの PUT セマンティクス)

## Context

各サービスの repository 層は 1 つの `XxxRepository` インターフェースと 1 つの `Repository`
struct に、読み取り (`List` / `Get`) と書き込み (`Create` / `Update`) を同居させていた。
handler は既に `xxx_read.go` / `xxx_write.go` にファイル分割され、責務として読み書きが
分かれているのに、依存する repository だけが両者を 1 つに束ねていた。

この非対称により次が起きていた:

- handler が読み取りハンドラからでも書き込みメソッドを持つインターフェース全体に依存し、
  「このハンドラは何を触るのか」が型から読み取れない。
- 将来 read 側だけ別経路 (キャッシュ / read replica) に差し替えたくなったとき、書き込みと
  同じ型に縛られて分離できない。

## Decision

**repository を CQRS の語彙で読み取り (Query) と書き込み (Command) に分割する。**

- 具象 struct を `XxxQuery` / `XxxCommand` に分ける (例: `MemberQuery` / `MemberCommand`)。
  どちらも sqlc 生成の `db.Querier` を保持する薄い委譲層である点は従来どおり。命名は
  `XxxQueryService` のような冗長な接尾辞を避け、リソース名 + `Query` / `Command` に統一する。
- パッケージ / ディレクトリ名から `repository` を外し `rdb` (relational database) にする。
  CQRS 分割後は「リポジトリ」という単一の集約はもう無く、`rdb` は「RDB アクセスの具象実装が
  集まる場所」を表す。ファイルも CQRS に合わせ `query.go` / `command.go` / `pool.go` に分ける。
- コンストラクタも `NewXxxQuery` / `NewXxxCommand` に分ける。`NewPool` (`pool.go`) は両者で共有する。
- 抽象 (インターフェース) は **利用側である handler パッケージ** に `Query` / `Command` として
  置く (consumer interface)。Go の「受け取りは interface、返すは具象」に従い、`rdb`
  パッケージからインターフェースを撤去する。
- handler も読み書きで型を分ける。`readHandler` は `query Query` だけ、`writeHandler` は
  `command Command` だけを持ち、それぞれ読み/書きエンドポイントのメソッドを実装する。公開する
  `Handler` は両者を埋め込み (`*readHandler` / `*writeHandler`)、メソッド昇格で単一の
  `api.ServerInterface` を満たす (codegen 由来のルータは 1 つの ServerInterface を要求するため、
  登録の単位としては合成型が要る)。`GetHealthz` は読み書きどちらの関心事でもないので合成
  `Handler` 自身に置く。これにより「読みハンドラから書き込みを呼べない」ことが型で保証される。
- DI (kessoku) は `Bind[handler.Query]` / `Bind[handler.Command]` で具象を束ねる。kessoku は
  具象→インターフェースの暗黙変換をしないため、`Bind` 対象の interface は **export が必須**。
  consumer interface だが export する。
- stub (`internal/stub`) は 4 メソッドを 1 つの型 (`MemberStub` 等、リソース名 + `Stub`) に
  実装したまま据え置く。構造的部分型により stub は `handler.Query` と `handler.Command` の
  両方を満たすので、読み/書きのどちらのテストでも同じ stub 型を使える。テスト用サーバは
  `newReadServer(query)` / `newWriteServer(command)` に分け、**読みテストは query だけ、
  書きテストは command だけ**を渡す (読みテストに書き込み依存を持ち込まない)。

## Consequences

- **読み書きの依存が型で分離される**: 読みハンドラは `Query` しか見えず、書きハンドラは
  `Command` しか見えない。handler の責務分割 (ADR-[[202606180902]] で触れた presentation 層の整理) と
  repository の形が揃う。
- **read 側を独立して差し替えられる余地ができる**: いまは ADR-[[202606170909]] の物理 DB 分割の範囲内で
  単一 pool を共有するが、`Query` 実装だけを別経路に差し替える拡張点ができた (現時点では
  read replica の必要性は未計測なので pool は分割しない)。
- **テストの注入点は維持**: `rdb` は実 DB 結合テスト、handler は stub の分岐網羅という
  ADR-[[202606180902]] の二段構えはそのまま。stub を分割しないので変更は最小。
- **注意点**: consumer interface を export する必要があるのは Go の idiom からはやや外れるが、
  kessoku の `Bind` が export 済みインターフェースを要求するための割り切りである。

## Alternatives considered

- **インターフェースだけ 2 分割し具象 struct は 1 つのまま**: 最小変更だが、1 つの struct が
  読み書き両方を実装し続けるため「read 側を差し替える」拡張点が生まれず、CQRS にする動機の
  半分しか満たさない。
- **インターフェースを具象と同じパッケージに置く (`rdb.MemberQuery` 等)**: 具象 struct
  を `MemberQuery` と名付けたいという要件と名前が衝突する。具象を `Query` / `Command` に縮める案も
  あったが、リソース名を冠した具象名 (`MemberQuery`) の方が DI やテストでの可読性が高い。
  consumer interface を handler 側に置くことで具象名を優先した。
- **read / write で pool を分割する**: 将来の read replica を見据えた物理分割。現時点で読み書きを
  別 DB に分ける必要性は計測されておらず、ADR-[[202606170909]] の customer/ops 分割で十分。`Query` /
  `Command` の型分離だけ先に入れ、pool 分割は必要になってから行う。
