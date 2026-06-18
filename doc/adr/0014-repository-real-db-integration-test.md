# ADR 0014: repository 層は実 DB 結合テストで検証する

- Status: Proposed
- Date: 2026-06-18
- Related: [[0012]] (customer/ops 物理 DB 分割) / [[0013]] (migration をサービスごとに分割)

## Context

Step 0 の骨格では handler / repository をいずれも **フェイク (`fakeQuerier`)** で検証していた。
repository 層は sqlc 生成の `Querier` に委譲するだけの薄い層なので、フェイクで通しても
「クエリの SQL が実 schema と噛み合うか」「`emit_empty_slices` で 0 件が nil でなく空スライス
で返るか」といった、**実 DB に当てて初めて分かる挙動** を検証できていなかった (issue #3)。

一方で結合テストの足場 ([[0013]]) は既に揃っていた:

- `compose.yaml` の `db-customer` (5432) / `db-ops` (5433)
- `scripts/migrate.sh` (goose をサービスごとに流す)
- CI の `server-integration` job (DB 起動 → migrate → `go test ./server/...`)

足りていなかったのは「実 DB に接続して検証するテスト本体」だけだった。

## Decision

**repository 層はフェイクをやめ、実 DB に接続する結合テストで検証する。**

- 接続先 DSN は環境変数で渡す ([[0013]] の `scripts/migrate.sh` と同じ規約)。
  - `order` / `payment` / `member` → `DATABASE_URL_CUSTOMER`
  - `product` / `shipping`        → `DATABASE_URL_OPS`
- **ビルドタグ (`//go:build integration`) は使わない。** 代わりに各テストの先頭で
  `testing.Short()` と DSN 未設定を見て `t.Skip` する。これにより:
  - `go test ./... -short` (per-service の `server` job) では skip
  - DSN 未設定のローカル `go test ./...` でも skip — DB が無くても緑
  - `server-integration` job / `mise run test:integration` では DSN を渡すので実行される
- テストは「table を `TRUNCATE ... RESTART IDENTITY` → seed → `List*` を呼んで検証」。
  ケースは [[test.md]] / [[go-test.md]] に従い 正常系 / 準正常系 / 異常系 でグルーピングする。
  異常系は「閉じた pool でクエリがエラーを伝播する」ことを確認する。
- CI の `server-integration` job の `test` step に DSN を `env:` で明示する (これが無いと
  全テストが skip して緑になり、検証が空になる)。

**ドメインロジック (handler 層) は引き続きスタブで単体テストする。**

- handler は JSON 整形・エラー → HTTP ステータス変換などのドメイン/HTTP ロジックを持ち、
  実 DB を必要としない。`internal/stub` の `Repo` で repository を差し替えて検証を続ける。
- これにより per-service の `server` job (`-short`, DB なし) でも handler の正常系・異常系
  (500 マッピング等) のカバレッジが保たれ、結合テストを skip してもカバレッジが崩れない。

## Consequences

- **実 SQL の回帰を検出できる**: schema 変更やクエリ修正が実 DB で検証される。
- **層ごとに責務が分かれる**: 「ドメインロジック = stub の単体テスト」「永続化 = 実 DB の
  結合テスト」と検証手段が層と対応する。
- **DB が無い環境でも `go test ./...` は緑**: skip するだけ。CI の per-service `server` job
  (`-short`) も DB 不要のまま。
- **結合テストの実行経路は 2 つ**: CI `server-integration` job と `mise run test:integration`。
  どちらも DB 起動 → migrate → DSN 付き `go test`。
- **注意点**: `server-integration` job で DSN を渡し忘れると「全 skip で緑」になり検証が
  空洞化する。DSN は workflow の `env:` と `scripts/migrate.sh` の default で二重化している。

## Alternatives considered

- **testcontainers-go (issue #3 の原文案)**: テスト内で Postgres コンテナを起動する。
  足場 ([[0013]]) の compose DB + migrate.sh が既にあり、CI も Docker 前提なので、
  依存を増やさず既存の DB を流用する方を採った。将来 compose に依らず単体で立てたく
  なったら再検討する。
- **`//go:build integration` ビルドタグで分離**: タグ付きファイルは IDE/補完/`go vet`
  の対象から外れやすく、「タグを付け忘れて常に skip」も起きやすい。`testing.Short()`
  + DSN 有無のランタイム判定なら通常ビルドに含まれ、skip 理由もログに出る。
- **フェイクを残して結合テストを併設**: 薄い委譲層に対する重複検証になり、フェイクの
  メンテだけが残る。repository では実 DB に寄せ、stub はドメインロジック (handler) に
  役割を限定した。
