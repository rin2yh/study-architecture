# ADR-202606180902: repository 層は実 DB 結合テストで検証する

- Status: Accepted
- Date: 2026-06-18
- Related: ADR-[[202606170909]] (customer/ops 物理 DB 分割) / ADR-[[202606180900]] (migration をサービスごとに分割)

## Context

Step 0 の骨格では handler / repository をいずれも **フェイク (`fakeQuerier`)** で検証していた。
repository 層は sqlc 生成の `Querier` に委譲するだけの薄い層なので、フェイクで通しても
「クエリの SQL が実 schema と噛み合うか」「`emit_empty_slices` で 0 件が nil でなく空スライス
で返るか」といった、**実 DB に当てて初めて分かる挙動** を検証できていなかった (issue #3)。

一方で結合テストの足場 (ADR-[[202606180900]]) は既に揃っていた:

- `compose.yaml` の `db-customer` (5432) / `db-ops` (5433)
- `scripts/migrate.sh` (goose をサービスごとに流す)
- CI の `server-integration` job (DB 起動 → migrate → `go test ./server/...`)

足りていなかったのは「実 DB に接続して検証するテスト本体」だけだった。

## Decision

**repository 層はフェイクをやめ、実 DB に接続する結合テストで検証する。**

- 接続先 DSN は環境変数で渡す (ADR-[[202606180900]] の `scripts/migrate.sh` と同じ規約)。
  - `order` / `payment` / `member` → `DATABASE_URL_CUSTOMER`
  - `product` / `shipping`        → `DATABASE_URL_OPS`
- **ビルドタグ (`//go:build integration`) は使わない。** 代わりに各テストの先頭で
  `testing.Short()` と DSN 未設定を見て `t.Skip` する。これにより:
  - `go test ./... -short` (per-service の `server` job) では skip
  - DSN 未設定のローカル `go test ./...` でも skip — DB が無くても緑
  - `server-integration` job / `mise run test:integration` では DSN を渡すので実行される
- テストは「table を `TRUNCATE ... RESTART IDENTITY` → seed → `List*` を呼んで検証」。
  ケースは [[test.md]] / [[go-test.md]] に従い 正常系 / 準正常系 / 異常系 でグルーピングする。
  異常系は「キャンセル済み context でクエリがエラーを伝播する」ことを確認する。
- skip 判定と接続は `server/internal/test/db` ヘルパー (`Open`) に集約する。
  ビルドタグを使わないので通常ビルドに含まれ、skip 理由もログに出る。
- CI の `server-integration` job の `test` step に DSN を `env:` で明示する (これが無いと
  全テストが skip して緑になり、検証が空になる)。

**handler 層 (presentation) も実 DB を通した結合テストで検証する。**

- handler は「ドメインロジック」ではなく **HTTP の入出力を担う presentation 層** である。
  そこで `HTTP → handler → rdb → 実 DB → JSON` の経路を通すフルスタックの結合テストを各サービスに
  足す。実 DB を使うべきなのは永続化層だけではない。
- 各エンドポイントごとに **`TestXxx` = 実 DB の正常系フルスタック / `TestXxxError` = stub の異常系**
  の 2 本に分ける (例: `TestGetOrder` は実 DB、`TestGetOrderError` は stub)。
  - **正常系 (happy path) は stub で代用しない。実 DB を通して** 「実 SQL と handler の入出力が
    噛み合うか」まで検証する。canned データを返す stub で 200/201 を作っても presentation の
    マッピングしか見ず、実経路を検証できないため。
  - **stub は異常系 (エラー注入) 専用**。エラー → 500 / 404 / 409 / 422 / 502 や validation 400 は
    実 DB では再現しづらく、stub でエラーを注入する方が確実に網羅できる。これらは DB 不要なので
    per-service の `server` job (`-short`) で実行され、DB が無くても presentation 層の分岐
    カバレッジが保たれる。
- つまり handler は **正常系 = 実 DB フルスタック + 異常系 = stub 注入** の二本立てで検証する。
  stub は「DB の代用 (正常系の肩代わり)」ではなく「異常系を注入するための点」として使う。
- 例外: 外部サービス呼び出し (order の checkout が product / payment ゲートウェイを叩く等) は、
  正常系でも実サービスを立てられないためゲートウェイのみ stub にする。command 側 (rdb) は実 DB を通す。

## Consequences

- **実 SQL の回帰を検出できる**: schema 変更やクエリ修正が実 DB で検証される。
- **検証手段が層と対応する**: rdb (永続化) は実 DB の結合テスト、handler (presentation) は
  正常系 = 実 DB フルスタック / 異常系 = stub 注入で検証する。正常系を stub で代用しない。
- **DB が無い環境でも `go test ./...` は緑**: 実 DB を要するテスト (正常系) は skip するだけ。CI の
  per-service `server` job (`-short`) は handler の異常系 stub テストで分岐カバレッジを保つ。
- **結合テストの実行経路は 2 つ**: CI `server-integration` job と `mise run test:integration`。
  どちらも DB 起動 → migrate → DSN 付き `go test`。
- **注意点**: `server-integration` job で DSN を渡し忘れると「全 skip で緑」になり検証が
  空洞化する。DSN は workflow の `env:` と `scripts/migrate.sh` の default で二重化している。

## Alternatives considered

- **testcontainers-go (issue #3 の原文案)**: テスト内で Postgres コンテナを起動する。
  足場 (ADR-[[202606180900]]) の compose DB + migrate.sh が既にあり、CI も Docker 前提なので、
  依存を増やさず既存の DB を流用する方を採った。将来 compose に依らず単体で立てたく
  なったら再検討する。
- **`//go:build integration` ビルドタグで分離**: タグ付きファイルは IDE/補完/`go vet`
  の対象から外れやすく、「タグを付け忘れて常に skip」も起きやすい。`testing.Short()`
  + DSN 有無のランタイム判定なら通常ビルドに含まれ、skip 理由もログに出る。
- **repository のフェイクを残して結合テストを併設**: 薄い委譲層に対する重複検証になり、
  フェイクのメンテだけが残る。repository は実 DB に寄せて `fakeQuerier` を撤去した。
- **handler も stub をやめて実 DB のみにする**: presentation 層を実 DB で通すのは正しいが、
  エラー → 500 などの分岐は実 DB では再現しづらく、かつ per-service の `server` job
  (`-short`, DB なし) で分岐カバレッジが落ちる。stub の単体テスト (分岐網羅) と実 DB の
  結合テストを併用する方が、網羅性と高速フィードバックを両立できる。
