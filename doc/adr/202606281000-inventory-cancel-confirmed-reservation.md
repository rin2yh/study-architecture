# ADR-202606281000: 確定済み予約のキャンセル戻しを予約行の cancelled_at で表す

- Status: Accepted
- Date: 2026-06-28
- Relates to: ADR-[[202606262000]] (予約の終端は相互排他の `*_at`), ADR-[[202606261702]] (order.cancelled の補償), ADR-[[202606261214]] (DB 制約で冪等化), GitHub #88

## Context

- #88 の補償で、未発送だが**決済確定済み** (`confirmed_at` セット) の注文をキャンセルすると在庫を戻す必要がある。
- ADR-[[202606262000]] は予約の終端を `CHECK (num_nonnulls(confirmed_at, released_at, expired_at) <= 1)` で相互排他にした。確定済み予約に `released_at` を立てられず、既存の `ReleaseReservationsByOrder` (未確定のみ解放) では確定済みの在庫を戻せない。

## Decision

確定済み予約の取り消しを、予約行の `cancelled_at` タイムスタンプで表す。

- **相互排他の終端 3 列の外側に `cancelled_at` を足す**。確定後の取り消しは終端状態の遷移ではなく確定済み行に乗る別事象なので、既存 CHECK を変えずに表せる。`CHECK (cancelled_at IS NULL OR confirmed_at IS NOT NULL)` で確定済み行にのみ立つことを DB で保証する。
- **利用可能在庫は `confirmed_at IS NOT NULL AND cancelled_at IS NULL` の確定予約だけが消費する**。導出は集計のまま、在庫数カラムは持たない (ADR-[[202606262000]])。
- **戻しは予約行に閉じる**。状態ガード付き `UPDATE ... WHERE cancelled_at IS NULL` で冪等にし、再配信での二重戻しを 0 行更新で吸収する (ADR-[[202606261214]] と同型)。未確定予約のキャンセルは従来どおり `released_at` で解放する。

## Consequences

- 予約のライフサイクル (予約→確定→取り消し) が 1 行に閉じ、別テーブルへの補償追記や相互参照が要らない。
- 相互排他の終端 3 列を緩めず確定済み戻しを表現できる (不変条件は不変)。
- `cancelled_at` は終端 3 列と排他でない。確定済み行に時刻が 2 つ立つが、各 `*_at` は NULL から一度だけ書かれ履歴は失わない。

## Alternatives considered

- **反対仕訳の補償 `stock_in` を追記**: 確定行の `-quantity` を消さず `+quantity` を足して相殺する案。`stock_ins` が入庫と戻しの 2 用途を持ち、戻し元参照列の命名・用途分割が要る。予約行に閉じる本決定の方が単純。
- **相互排他 CHECK を緩め確定済みにも `released_at` を許す**: 終端 3 列の「1 つだけ」という不変条件を崩す。`cancelled_at` を別列にすれば不変条件を保ったまま表せる。
