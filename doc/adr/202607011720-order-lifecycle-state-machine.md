# ADR-202607011720: order のライフサイクルを状態機械 + 追記履歴で表す

- Status: Accepted (語彙は ADR-[[202607020305]] で payment_pending / failed を追加)
- Date: 2026-07-01
- Relates to: ADR-[[202607011621]] (マイクロサービス移行。本 ADR はその具体化タスク 1), ADR-[[202606261702]] (cancel 補償と現行のキャンセル可否判定), ADR-[[202606261700]] (inventory 台帳の append-only 思想), ADR-[[202606261212]] (Outbox 列。履歴テーブルとは別物), ADR-[[202606211200]] (payment.settled / shipment イベント = 前進トリガ), ADR-[[202606250159]] (traceparent 伝播), ADR-[[202606180902]] (実 DB 結合テスト), ADR-[[202606190903]] (repository CQRS), GitHub #98 (親), #96

## Context

- `orders.status` は `text NOT NULL` で CHECK も enum も無く、`CreateOrder` / `UpdateOrder` は `req.Status` を素通しで保存する (検証なし)。実際に書かれる値は checkout の `"confirmed"` と `CancelOrder` の `"cancelled"` の 2 つだけ。
- 遷移規則は `CancelOrder` (`server/order/internal/rdb/command.go`) の string switch だけにある: `shipped`→キャンセル不可 / `cancelled`→冪等 noop。
- しかし **order 内に status を `shipped` へ進める書き手が存在しない** (計測済み)。paid / shipped の真実は payment / shipping の別 status に閉じ、order へ反映されない。order の状態機械は到達しない状態を守っている宙吊りの形。
- #96 が status の履歴化を先送りしており、キャンセル可否・注文履歴 UI が読む「注文の今」を単一の裸カラムに委ねている。

## Decision

order のライフサイクルを **明示的な状態機械**として定義し、**現在状態列 + 追記型の遷移履歴テーブル**で保存、遷移規則は **アプリ層の FSM**で強制する。フル語彙 (paid / shipped を含む) を order に集約する。

- **状態語彙**: `placed → paid → shipped` の前進系列と、終端 `cancelled`。現行の `"confirmed"` は `placed` へリネームする。`shipped` と `cancelled` が終端。中間状態 `fulfilling` は置かず、必要になれば後で足す。
- **遷移辺**: `∅→placed` (checkout)、`placed→paid` (payment.settled 購読)、`paid→shipped` (shipment 出荷イベント購読)、`{placed,paid}→cancelled` (補償。`shipped` からは不可 = 現行 `ErrNotCancellable` を一般化)。
- **保存**: `orders.status` を現在状態として残し O(1) で読む。値域を DB の CHECK で語彙へ制約する。追記専用 `order_status_transitions (order_id, from_status, to_status, cause, traceparent, occurred_at)` に全遷移を記録する。`cause` は遷移の引き金 (イベント名や actor、例 `payment.settled` / `customer-cancel`)、`traceparent` はその遷移を引き起こした分散トレース (ADR-[[202606250159]]) との紐付け。履歴と監査証跡で #96 の履歴化要望を満たす。
- **強制**: 許可された `(from,to)` 集合を Go 側 FSM に集約する。遷移 API は `GetOrderForUpdate` で行ロック → FSM で `(from,to)` を検証 → `status` UPDATE と `order_status_transitions` INSERT を **1 tx** で確定 (現在列と履歴の整合を担保。現行 `CancelOrder` の `GetForUpdate→switch→遷移` を一般化)。並行は悲観ロック (`FOR UPDATE`) で直列化し version 列は持たない (現行踏襲)。
- **`paid` / `shipped` への前進はイベント購読で行う**が、その購読者の実装とオーケストレーション方式 (choreography / orchestration) は **タスク 2 の別 ADR** に切る (1 ADR=1 決定)。本 ADR は FSM の定義・保存・強制までを確定する。

## Consequences

- 現在列と履歴の二重書きの整合を 1 tx で担保する責務が order の command 層に増える (トレードオフ)。
- FSM を Go に集約するため遷移規則をテストで被覆できる (ADR-[[202606180902]] の実 DB 結合テストで各辺を確認)。
- 本 ADR 完了時点で実遷移するのは `placed` / `cancelled` のみ。`paid` / `shipped` は語彙・辺として定義済みで購読配線待ち = 移行の既知の途中状態 (ADR-[[202607011621]] の hybrid を許容する原則どおり)。
- checkout 前失敗の巻き戻し (`abandonCheckout` / `DeleteOrder`) は当面現行のまま。サーガ化での見直しはタスク 2。
- `order_status_transitions` は **監査履歴**であり、イベント発行の Outbox (ADR-[[202606261212]] / #96 で ADR-202606300600 へ移行中) とは別物。混同して発行を履歴テーブルに相乗りさせない。

## Alternatives considered

- **イベントソーシング (状態を導出)**: 監査最強で inventory 台帳 (ADR-[[202606261700]]) と同思想だが、projection / snapshot と結果整合の読みを持ち込む。現在列の O(1) 読みと既存の同期的な注文履歴照会 (`ListOrdersByMember`) に対しオーバー。現在列 + 履歴のハイブリッドで読みの安さを採る。
- **単一 enum 列のみ (履歴なし)**: 最小だが履歴が残らず #96 の履歴化要望に未達。
- **DB トリガで遷移も強制**: 不正遷移を DB で拒否できるが、遷移規則 (業務条件) が SQL に散りテスト・可視性が落ちる。このリポジトリにトリガの前例が無い。FSM は Go、DB は値域 (CHECK) のみに分ける。
- **paid / shipped を order に集約せず各サービスへ委ねる**: order 単体でライフサイクルが読めず、注文履歴 UI とキャンセル可否判定が多サービス照会になる。フル語彙を order へ集約する (語彙決定どおり)。
