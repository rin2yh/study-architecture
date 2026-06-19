# ADR 0017: repository を CQRS で Query / Command に分割する

- Status: Accepted
- Date: 2026-06-19
- Related: [[0014]] (repository は実 DB 結合テストで検証) / [[0015]] (更新エンドポイントの PUT セマンティクス)

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
- コンストラクタも `NewXxxQuery` / `NewXxxCommand` に分ける。`NewPool` は両者で共有する。
- 抽象 (インターフェース) は **利用側である handler パッケージ** に `Query` / `Command` として
  置く (consumer interface)。Go の「受け取りは interface、返すは具象」に従い、repository
  パッケージからインターフェースを撤去する。handler は `query Query` / `command Command` の
  2 フィールドを持ち、読みハンドラは `query`、書きハンドラは `command` だけを使う。
- DI (kessoku) は `Bind[handler.Query]` / `Bind[handler.Command]` で具象を束ねる。kessoku は
  具象→インターフェースの暗黙変換をしないため、`Bind` 対象の interface は **export が必須**。
  consumer interface だが export する。
- stub (`internal/stub`) は従来どおり 4 メソッドを 1 つの `Repo` に実装したまま据え置く。
  構造的部分型により `Repo` は `handler.Query` と `handler.Command` の両方を満たすので、
  テストでは同じ stub を query / command の両引数に渡せる。

## Consequences

- **読み書きの依存が型で分離される**: 読みハンドラは `Query` しか見えず、書きハンドラは
  `Command` しか見えない。handler の責務分割 ([[0014]] で触れた presentation 層の整理) と
  repository の形が揃う。
- **read 側を独立して差し替えられる余地ができる**: いまは [[0012]] の物理 DB 分割の範囲内で
  単一 pool を共有するが、`Query` 実装だけを別経路に差し替える拡張点ができた (現時点では
  read replica の必要性は未計測なので pool は分割しない)。
- **テストの注入点は維持**: repository は実 DB 結合テスト、handler は stub の分岐網羅という
  [[0014]] の二段構えはそのまま。stub を分割しないので変更は最小。
- **注意点**: consumer interface を export する必要があるのは Go の idiom からはやや外れるが、
  kessoku の `Bind` が export 済みインターフェースを要求するための割り切りである。

## Alternatives considered

- **インターフェースだけ 2 分割し具象 struct は 1 つのまま**: 最小変更だが、1 つの struct が
  読み書き両方を実装し続けるため「read 側を差し替える」拡張点が生まれず、CQRS にする動機の
  半分しか満たさない。
- **インターフェースを repository パッケージに置く (`repository.MemberQuery` 等)**: 具象 struct
  を `MemberQuery` と名付けたいという要件と名前が衝突する。具象を `Query` / `Command` に縮める案も
  あったが、リソース名を冠した具象名 (`MemberQuery`) の方が DI やテストでの可読性が高い。
  consumer interface を handler 側に置くことで具象名を優先した。
- **read / write で pool を分割する**: 将来の read replica を見据えた物理分割。現時点で読み書きを
  別 DB に分ける必要性は計測されておらず、[[0012]] の customer/ops 分割で十分。`Query` /
  `Command` の型分離だけ先に入れ、pool 分割は必要になってから行う。
