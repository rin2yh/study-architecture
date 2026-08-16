# ADR-202608160810: 先に届いた order.cancelled を shipments 行として残す

- Status: Accepted
- Date: 2026-08-16
- Relates to: ADR-[[202608160800]] (順不同の契約), ADR-[[202606261702]] (order.cancelled の補償), ADR-[[202606261214]] (DB 制約で冪等化), GitHub #107

## Context

- 順不同の前提は ADR-[[202608160800]]。
- 実測: `order.cancelled` を先に処理すると `CancelShipmentForOrder` が 0 行更新で終わり、後から
  届いた `payment.settled` が `preparing` の shipment を作った。**キャンセル済み注文に配送枠が
  残る**。inventory は解放済み予約を確定対象から外すため、同じ順序でも壊れなかった。

## Decision

shipping は `order.cancelled` を受けたとき、shipment が未手配でも `shipments` に
`status = 'cancelled'` の行を立てる (`INSERT ... ON CONFLICT (order_id) DO UPDATE`)。

- `order_id` は既に UNIQUE なので、注文ごとの終端状態はこの 1 行に載せられる。後着の
  `payment.settled` は既存の `ON CONFLICT DO NOTHING` にそのまま弾かれ、**どちらの順で届いても
  同じ結果に収束する**。
- 判定を DB の 1 文に閉じ、注文状態の読み直し (check-then-act) を持たない (ADR-[[202606261214]] と同型)。

## Consequences

- 配送手配前にキャンセルされた注文が、宛先が空の `cancelled` 行として一覧に出る。`shipments` は
  「手配された配送」ではなく「注文ごとの配送の顛末」を表すテーブルになる。
- 順不同耐性がキューの設定でなく shipping 自身のスキーマに載るため、ブローカを替えても効く。
- 発送済み (`shipped`) の行は `WHERE status = 'preparing'` のガードで従来どおり書き換えない。

## Alternatives considered

- **専用テーブル `cancelled_orders` を足す**: `shipments` の意味を保てるが、注文ごとの終端状態が
  2 テーブルに割れ、手配可否の判定に結合が要る。UNIQUE 制約が既にある 1 行に寄せる方が単純。
- **`payment.settled` 処理で order の状態を読んでガードする**: 既に宛先取得で order を引いており
  追加の往復は不要だが、読んでから INSERT するまでの間にキャンセルされると素通りする。
- **順序保証を導入する**: ADR-[[202608160800]] で不採用。
