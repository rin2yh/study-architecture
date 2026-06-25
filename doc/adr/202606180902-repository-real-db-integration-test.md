# ADR-202606180902: repository 層は実 DB 結合テストで検証する

- Status: Accepted (本文の「異常系 = stub 注入」はエラー注入という検証手段の軸を指す。テストの
  ケース分類 (4xx=準正常系 / 5xx=異常系) は ADR-[[202606211520]] で更新)
- Date: 2026-06-18
- Related: ADR-[[202606170909]] (customer/ops 物理 DB 分割) / ADR-[[202606180900]] (migration をサービスごとに分割)

## Context

- Step 0 では repository も handler もフェイク (`fakeQuerier`) で検証していた。repository は
  sqlc 生成の `Querier` に委譲するだけの薄い層で、「SQL が実 schema と噛み合うか」「`emit_empty_slices`
  で 0 件が空スライスで返るか」など実 DB でしか分からない挙動を検証できていなかった (issue #3)。
- 結合テストの足場 (compose の DB / `scripts/migrate.sh` / CI の `server-integration` job) は
  ADR-[[202606180900]] で既に揃っており、足りないのは実 DB に当てるテスト本体だけだった。

## Decision

**repository 層はフェイクをやめ、実 DB に接続する結合テストで検証する。**

- 接続先 DSN は環境変数で渡す (ADR-[[202606180900]] と同じ規約)。
- **ビルドタグは使わず**、各テスト先頭で `testing.Short()` と DSN 未設定を見て `t.Skip` する。
  これで `-short` / DSN 未設定のローカルでは skip して緑、`server-integration` job では DSN を
  渡して実行される。why はビルドタグを避ける理由 (下記 Alternatives) と同根。
- skip 判定と接続は `server/internal/test/db` ヘルパーに集約する (通常ビルドに含まれ、skip 理由も
  ログに出る)。
- CI `server-integration` job の test step に DSN を `env:` で明示する。これが無いと全 skip で緑に
  なり検証が空洞化するため。

**handler 層 (presentation) も実 DB を通した結合テストで検証する。**

- handler はドメインロジックではなく HTTP 入出力の presentation 層。`HTTP → handler → rdb → 実 DB
  → JSON` のフルスタックを通す。実 DB を使うのは永続化層だけではない、が決め手。
- 各エンドポイントを **正常系 = 実 DB フルスタック / 異常系 = stub 注入** の二本立てにする。
  - 正常系を stub で代用しないのは、canned データ stub では presentation のマッピングしか見えず、
    実 SQL と handler の入出力が噛み合うかを検証できないため。
  - stub は異常系専用。エラー → 500/404/409/422/502 や validation 400 は実 DB で再現しづらく、
    stub 注入の方が確実に網羅でき、DB 不要なので `-short` の `server` job でも分岐を保てる。
- 例外: 外部サービス呼び出し (order checkout が product/payment ゲートウェイを叩く等) は正常系でも
  実サービスを立てられないため、ゲートウェイのみ stub にする。command 側 (rdb) は実 DB を通す。

## Consequences

- schema/クエリ変更による実 SQL の回帰を検出できる。
- 検証手段が層に対応する: rdb は実 DB、handler は正常系 = 実 DB / 異常系 = stub 注入。
- 実 DB を要するテストは skip するだけなので、DB が無い環境でも `go test ./...` は緑。`-short` の
  `server` job は handler の異常系 stub で分岐カバレッジを保つ。
- 結合テストの実行経路は CI `server-integration` job と `mise run test:integration` の 2 つ。
- 注意: `server-integration` で DSN を渡し忘れると全 skip で緑になり検証が空洞化する。workflow の
  `env:` と `scripts/migrate.sh` の default で二重化している。

## Alternatives considered

- **testcontainers-go (issue #3 原案)**: テスト内で Postgres を起動する。compose DB + migrate.sh が
  既にあり CI も Docker 前提なので、依存を増やさず既存 DB を流用した。compose に依らず単体で立てたく
  なったら再検討。
- **`//go:build integration` ビルドタグで分離**: タグ付きファイルは IDE/補完/`go vet` から外れやすく
  「タグ付け忘れで常に skip」も起きやすい。`testing.Short()` + DSN 有無のランタイム判定なら通常
  ビルドに含まれ、skip 理由もログに出る。
- **repository のフェイクを残して結合テストを併設**: 薄い委譲層への重複検証になりフェイクのメンテだけ
  残る。実 DB に寄せて `fakeQuerier` を撤去した。
- **handler も stub をやめ実 DB のみにする**: エラー分岐は実 DB で再現しづらく、`-short` (DB なし) の
  `server` job で分岐カバレッジが落ちる。stub の分岐網羅 + 実 DB の結合テスト併用の方が網羅性と
  高速フィードバックを両立できる。
