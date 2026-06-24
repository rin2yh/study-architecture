# ADR-202606240821: server-unit と server-integration を単一 server-test job に集約する

- Status: Accepted
- Date: 2026-06-24
- Supersedes: ADR-[[202606180902]] (CI job 二段構えの部分), ADR-[[202606220716]] (server job 構成の部分)
- Relates to: ADR-[[202606190902]] (テンプレート DB クローン並列), ADR-[[202606211520]] (テストのケース分類), [[ci.md]] (CI 規約)

## Context

各量子の CI (`ci-customer.yml` / `ci-backoffice.yml`) は server テストを 2 つの job に分けていた:

- `server-unit` (matrix per-service): `go test ./server/$svc/... ./server/internal/... -short`
- `server-integration`: DSN 付きで `go test <量子全パッケージ> ./server/internal/...` (**`-short` なし**)

ところが Go の `-short` は「結合テストを `skip` させる」一方向フラグでしかなく (skip 判定は
`skip.Short` = `testing.Short()`、ADR-[[202606180902]])、「ユニットだけ実行する」逆向きの指定は
無い。結果、**`-short` なしの `server-integration` は結合テストに加えてユニットテストも全部
実行**しており、`server-unit` と実行が丸かぶりだった。

計測 (顧客系量子, 2026-06-24 時点):

- テスト関数 計 78 (order 25 / payment 18 / member 27 / internal 8)。うちユニットのみのファイル 13。
- `server/internal/...` のユニットは `server-unit` の matrix `[order, payment, member]` で **3 回** +
  `server-integration` で **1 回** = 最大 **4 回**実行されていた。

ADR-[[202606180902]] は「rdb は実 DB 結合 / handler は 正常系=実 DB・異常系=stub」という
**検証手段の二段構え**を定めたもので、これは今も有効である。一方で **CI job を 2 つに割る**
理由 (高速な DB 不要フィードバック) は、`server-integration` が PR ごとに必ず走り (e2e の
`needs`) かつユニットを内包する以上、別 job を維持するほどの利得が無かった。

## Decision

**server テストの CI job を 1 本 (`server-test`) に集約し、`server-unit` を撤去する。**

- `server-test` は `-short` を付けず、`go test <量子全パッケージ> ./server/internal/...` を 1 回回す。
  これでユニット + 実 DB 結合が 1 ランで揃い、ユニットの二重実行が消える。
- `server-unit` が持っていた `go fmt` チェックと `go vet` は `server-test` の冒頭 step に移し、
  DB 起動より前に置いて fast fail を保つ。
- カバレッジは集約ラン 1 本で計測し、量子単位の 60% ゲートを維持する (per-service ゲートと
  per-service コメントは廃止)。コメント header は `coverage-server-test-<量子>`。
- e2e の `needs` は `server-test` に張り替える。`server-test` 自身は `needs` を持たない
  (旧 `server-integration` の `needs: [server-unit]` は集約で不要)。

## Consequences

- **ユニットの二重実行が消える**: 旧構成で最大 4 回走っていた `server/internal/...` のユニットが
  1 回になる。CI の重複実行とカバレッジ二重計上が解消する。
- **検証手段の二段構えは不変**: rdb=実 DB / handler=正常系実 DB・異常系 stub という
  ADR-[[202606180902]] の Decision はテスト側でそのまま生きる。今回変えたのは「CI job の分け方」だけ。
- **per-service の粒度を失う**: ユニットだけの 60% ゲート・サービス別カバレッジコメントは無くなり、
  量子単位の集約カバレッジに一本化される。学習用リポで required check を運用しない
  (ADR-[[202606220716]]) ため、ゲート粒度低下の実害は小さい。
- **DB 不要の高速フィードバックを失う**: server テストは常に DB 起動を伴う。client のみ変更の PR でも
  broad fan-out (ADR-[[202606220716]]) で `server-test` が回ると DB を立てる。`server-integration` が
  元から毎 PR 走っていたため増分は小さい。fmt/vet は DB 起動前 step なので速報性は保たれる。
- **job 名の変更**: `server-unit-*` / `server-integration` は消え `server-test` になる。required check を
  運用していない (ADR-[[202606220716]]) ので merge ブロックは発生しない。旧 sticky コメント
  (`coverage-server-unit-*` / `coverage-server-integration-*`) は既存 PR に残骸として残るが無害。

## Alternatives considered

- **`server-integration` 側でユニットを除外する (逆フラグ / `-run` 限定)**: `RUN_INTEGRATION` で
  ユニットを `t.Skip` させる逆向きフラグや、`-run` で結合テスト名に限定する案。二重実行は消えるが、
  全テストに skip 分岐や命名規約を増やすことになり、ADR-[[202606180902]] が避けたビルドタグと
  同種の「付け忘れで検証が空洞化」リスクを別の形で持ち込む。job を 1 本にする方が単純。
- **`server-unit` だけ残し `server-integration` を消す**: 速いが実 DB 結合 (本来の検証主目的) を
  失う。逆である。集約するなら結合を内包する側 (`-short` なしラン) を残す。
- **internal の多重実行だけ潰す (internal を独立 job 化)**: 最も無駄な internal 4 回は減るが、
  量子サービス側のユニット二重実行 (server-unit と server-integration) は残る。job 数も増える。
  根本である「`-short` なしランがユニットを内包する」点に対しては集約が素直。
- **現状維持 + ADR 明記のみ**: 二重実行を「高速フィードバック vs 集約カバレッジ」のコストとして
  許容する案。だが集約カバレッジ側 (`server-integration`) が毎 PR 走りユニットを内包する以上、
  別 job の利得が無く、無駄を残す積極的理由が無かった。
