# ADR-202607011200: handler の port を domain/DTO 型に統一する (db.* を露出しない)

- Status: Accepted
- Date: 2026-07-01
- Relates to: ADR-[[202606190903]] (CQRS Query/Command) / ADR-[[202606170901]] (codegen-first) / ADR-[[202606261704]] (order スナップショット)

## Context

handler の port である `Query` / `Command` (ADR-[[202606190903]]) に sqlc 生成型 (`db.*` 行型・`db.*Params`) が入力・返り値の両方で露出していた。全 6 サービス共通。port は handler を repository 実装から切り離す抽象なのに、その署名が永続化 codegen 出力に結合していた。

- DB スキーマ変更が handler まで波及する。行型は全カラムを含むため、API に出さない内部列 (outbox 列・snapshot 列・`idempotency_key` 等) も port を通って露出する。outbox 列の増減 (#96 / #97) がそのまま interface の型に現れていた。
- 入力の一部は既に DTO 化されていた (`rdb.CheckoutLine` / `rdb.CheckoutAddress` / `rdb.ReserveLine` / `rdb.PaymentUpdate`) が、返り値と多くの params は `db.*` のままで、境界のポリシーが非対称だった。

## Decision

**port は `db.*` を露出せず、`rdb` パッケージに手書きした domain/DTO 型で入出力する。db ↔ DTO の変換は `rdb` 層に閉じ込める。**

- 返り値は行型ごとに DTO を定義 (`rdb.Payment` / `rdb.Order` / `rdb.OrderItem` ...)。API に出さない内部列は載せず、`pgtype` は標準型 (`time.Time` 等) へ寄せる。
- 入力は `db.*Params` を DTO へ置換 (`rdb.PaymentCreate` / `rdb.OrderCreate` / `rdb.OrderUpdate` ...)。
- 変換は `rdb` の `toXxx` ヘルパに集約し、`db.*Params` の組み立てと行型→DTO の変換をメソッド内で行う。handler・stub は `db` を import しない。
- DTO の置き場所は `rdb`。既存の `rdb.CheckoutLine` 等と同じ方向で、handler → rdb → db の依存を保ち循環を避ける (別 `domain` パッケージは新設しない)。

決め手: schema 変更の影響を rdb のマッピングに閉じ込め、port を codegen から独立させる。既に一部で採っていた入力 DTO 化を返り値まで広げ、境界のポリシーを対称にする。

## Consequences

- schema・sqlc 再生成の影響が `rdb` のマッピングで止まり、interface・handler・stub へ波及しない。
- 内部専用列が port の型に載らない。DTO に足さない限り handler から見えない。
- 代償はマッピングのボイラープレート (`toXxx` と入力 DTO の組み立て)。手書き量は増えるが、影響範囲を局所化する対価とする。
- codegen-first (ADR-[[202606170901]]) の「手書きはグルーに絞る」とはトレードオフ。sqlc 型は `rdb` 内に留め、境界にはグルーとしての変換を1枚挟む形にした。

## Alternatives considered

- **`db.*` を port に通す方針を明文化する**: マッピング層を持たない割り切り。ボイラープレートは無いが、schema 変更が handler まで波及する構造と内部列の露出が残る。issue #100 が解消したかった非対称そのものなので採らない。
- **DTO を専用 `domain` パッケージに置く**: 概念的には綺麗だが、handler・rdb 双方から import する新パッケージが要り、既存の `rdb.CheckoutLine` 配置とも二重になる。既存慣行に合わせ `rdb` に置いた。
