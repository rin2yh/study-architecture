# ADR-202606180902: repository 層は実 DB 結合テストで検証する

- Status: Accepted (本文の「異常系 = stub 注入」はエラー注入という検証手段の軸を指す。テストの
  ケース分類 (4xx=準正常系 / 5xx=異常系) は ADR-[[202606211520]] で更新)
- Date: 2026-06-18
- Related: ADR-[[202606170909]] (customer/ops 物理 DB 分割) / ADR-[[202606180900]] (migration をサービスごとに分割)

## Context

- Step 0 では repository もフェイク (`fakeQuerier`) で検証しており、「SQL が実 schema と噛み合うか」
  「`emit_empty_slices` で 0 件が空スライスで返るか」など実 DB でしか分からない挙動が抜けていた (issue #3)。
- 結合テストの足場は ADR-[[202606180900]] で揃っており、足りないのはテスト本体だけだった。

## Decision

**repository 層はフェイクをやめ、実 DB に接続する結合テストで検証する。**

- **ビルドタグは使わず**、各テストで `testing.Short()` と DSN 未設定を見て `t.Skip` する。通常ビルドに
  含まれ、skip 理由もログに出るため (不採用理由は下記)。判定と接続は `server/internal/test/db` に集約。
- CI の該当 job に DSN を明示的に渡す。渡し忘れると全 skip で緑になり、検証が空洞化するため。

**handler 層 (presentation) も実 DB を通した結合テストで検証する。**

- handler は HTTP 入出力の presentation 層なので、`HTTP → handler → rdb → 実 DB → JSON` を通す。
  実 DB を使うのは永続化層だけではない、が決め手。
- 各エンドポイントを **正常系 = 実 DB フルスタック / 異常系 = stub 注入** の二本立てにする。canned データの
  stub では実 SQL と handler の入出力が噛み合うかを見られず、逆にエラー経路は実 DB で再現しづらいため。
- 例外: 外部サービス呼び出しは正常系でも立てられないので、ゲートウェイのみ stub にする。

## Consequences

- schema / クエリ変更による実 SQL の回帰を検出できる。
- DB が無い環境でも skip されるだけなので `go test ./...` は緑のまま。`-short` の job は handler の
  異常系 stub で分岐カバレッジを保つ。
- DSN の渡し忘れが「緑だが何も検証していない」を生む。workflow の `env:` と `scripts/migrate.sh` の
  default で二重化しているが、構造的な弱点として残る。

## Alternatives considered

- **testcontainers-go (issue #3 原案)** — テスト内で Postgres を起動する。compose DB と migrate.sh が既にあり
  CI も Docker 前提なので、依存を増やさず既存 DB を流用した。compose に依らず立てたくなったら再検討。
- **`//go:build integration` で分離** — タグ付きファイルは IDE・`go vet` から外れやすく、タグ付け忘れで
  常に skip も起きる。
- **repository のフェイクを残して結合テストを併設** — 薄い委譲層への重複検証で、フェイクのメンテだけ残る。
- **handler も実 DB のみにする** — エラー分岐が再現しづらく、DB なしの job で分岐カバレッジが落ちる。
