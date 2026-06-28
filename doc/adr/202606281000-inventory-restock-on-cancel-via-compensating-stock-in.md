# ADR-202606281000: 確定済み予約のキャンセル戻しを補償 stock_in (反対仕訳) で表す

- Status: Accepted
- Date: 2026-06-28
- Relates to: ADR-[[202606262000]] (予約の終端は相互排他の `*_at`), ADR-[[202606261702]] (order.cancelled の補償), ADR-[[202606261214]] (DB 制約で冪等化), GitHub #88

## Context

- #88 の補償で、未発送だが**決済確定済み** (`confirmed_at` セット) の注文をキャンセルすると在庫を戻す必要がある。
- ADR-[[202606262000]] は予約の終端を `CHECK (num_nonnulls(confirmed_at, released_at, expired_at) <= 1)` で相互排他にした。確定済み予約に `released_at` を立てられず、既存の `ReleaseReservationsByOrder` (未確定のみ解放) では確定済みの在庫を戻せない。

## Decision

確定済み予約の戻しは `stock_ins` への補償追記 (+quantity) で表す。

- **未確定は従来どおり `released_at` で解放、確定済みは入庫として相殺**する。確定行の `-quantity` を消さず、対になる `+quantity` を足して集計を戻す (在庫数を導出で保つ ADR-[[202606262000]] の不変条件を崩さない)。
- **`stock_ins.reservation_id` で戻し元の予約を辿れるようにし**、同列の部分ユニークを再戻しの冪等キーにする。再配信は `ON CONFLICT DO NOTHING` で 0 行に収束する (ADR-[[202606261214]] と同型)。通常の入庫は NULL。

## Consequences

- 相互排他 CHECK を緩めずに確定済みの戻しを表現でき、ADR-[[202606262000]] の終端モデルを変えない。
- 在庫変動の履歴に「戻し」が独立行として残り、どの予約由来かを `reservation_id` で追える。
- `stock_ins` が純粋な入庫だけでなく補償戻しも持つ。意味の混在は `reservation_id` の有無で判別する。

## Alternatives considered

- **相互排他 CHECK を緩め確定済みにも `released_at` を許す**: 行内更新で完結するが ADR-[[202606262000]] の「終端は 1 つ」という不変条件を崩し、集計 (`confirmed_at IS NOT NULL`) も `released_at IS NULL` 条件付きへ波及する。
- **`reason` 列で戻しを区別**: 文字列の語彙管理が増え、トレーサビリティ (どの予約の戻しか) を持てない。`reservation_id` 参照なら出所と冪等キーを兼ねられる。
