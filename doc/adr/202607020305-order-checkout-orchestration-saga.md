# ADR-202607020305: checkout を order オーケストレーション型サーガにし前進フローを非同期化する

- Status: Accepted
- Date: 2026-07-02
- Relates to: ADR-[[202607011621]] (マイクロサービス移行。本 ADR は具体化タスク 2), ADR-[[202607011720]] (order FSM。本 ADR が語彙を拡張), ADR-[[202606261702]] (order.cancelled 補償コレオグラフィ), ADR-[[202606211200]] (payment.settled イベント), ADR-[[202606261212]] (Outbox。#96 で ADR-202606300600 へ移行中), ADR-[[202606261214]] (冪等), ADR-[[202606261216]] (致命/縮退), ADR-[[202606250159]] (traceparent), GitHub #98 (親), #127 (本タスクの issue), #104 (メッセージング信頼性 = 本サーガの前提)

## Context

- checkout (`server/order/internal/handler/order_write.go`) は order がハンドラ内で同期 HTTP (`inventory.Reserve` → `payment.CreatePayment`) を束ねる揮発オーケストレーション。payment / inventory 障害が checkout を同期的に落とし、量子の可用性が独立しない (ADR-[[202607011621]] が挙げた乖離の実例)。補償は `abandonCheckout` でインライン。
- タスク 1 (ADR-[[202607011720]]) で order を FSM 化したが、`paid` / `shipped` への配線は本タスクへ先送りした。
- 既存の非同期: `payment.settled`→shipping (ADR-[[202606211200]])、`order.cancelled`→補償 (ADR-[[202606261702]])。order は `payment.settled` を未購読。

## Decision

checkout を **order をオーケストレータとするサーガ**にし、前進フロー (予約・決済) を非同期化する。order の FSM をサーガ状態として駆動する。

- **checkout は即時受理**: `POST /checkout` は注文を `placed` で作成し orderId を即返す (202 相当)。在庫不足・決済失敗は同期応答で返さず、サーガが order を失敗終端 `rejected` へ落として可視化する (理由は履歴の `cause`)。
- **order.status をサーガ状態として拡張**: タスク 1 の `placed → paid → shipped` / `cancelled` に、中間 `payment_pending` と失敗終端 `rejected` を足す (ADR-[[202607011720]] の語彙を追記更新)。
  - 遷移辺: `∅→placed` (受理)、`placed→payment_pending` (予約成功購読)、`placed→rejected` (在庫不足購読・補償不要)、`payment_pending→paid` (`payment.settled` 購読)、`payment_pending→rejected` (決済失敗購読 → 予約解放の補償)、`paid→shipped` (出荷購読)、`{placed,payment_pending,paid}→cancelled` (顧客取消。`shipped` からは不可)。
- **調整はコマンド / イベントで非同期に**: order は予約・決済を同期 HTTP でなく非同期に依頼し、各サービスの結果イベント (`reserved` / 在庫不足, `payment.settled` / 決済失敗, 出荷) を購読して FSM を進める。既存の Redis Streams + Outbox 経路 (ADR-[[202606261212]]) に相乗りする。
- **補償はコレオグラフィを再利用**: 失敗遷移 (`payment_pending→rejected` 等) は `order.cancelled` 相当のイベントで inventory 解放などを後追いする (ADR-[[202606261702]])。インライン `abandonCheckout` / `DeleteOrder` は廃止し、失敗も行を残して履歴に記録する。
- **冪等**: 各遷移は購読イベントに対し FSM の `(from,to)` ガードで冪等 (二重配信を 0 行に。ADR-[[202606261214]] と同型)。checkout 発番の冪等キー (ADR-[[202606261214]]) は継続。

## Consequences

- payment / inventory 障害で checkout が即失敗しなくなる → 量子の可用性が独立し主駆動特性 (ADR-[[202607011621]]) を満たす。
- 同期の即時フィードバック (在庫不足 409) が消え、クライアントは order 状態のポーリング / 通知で結果を知る。UI / BFF 側の結果取得は別タスク。
- order がイベント購読者・コマンド発行者になり、order にコンシューマ (予約結果 / `payment.settled` / 出荷) が増える。
- サーガ状態を order.status に集約するため FSM が太る (`payment_pending` / `rejected` 追加)。タスク 1 ADR を追記更新する。
- 分散の失敗様態 (途中失敗・重複配信) が顕在化 → 冪等と補償で吸収。traceparent 履歴 (ADR-[[202606250159]]) がデバッグの生命線。
- **前提**: 本サーガは購読イベントの再配送 (at-least-once) を当て込むが、現状の consumer は `XReadGroup ">"` で失敗イベントを再配送しない (#104)。メッセージング信頼性 (Step 6: XAUTOCLAIM 引き取り・DLQ) を先行させないと、途中失敗した order が前進せず滞留する。#104 を前提に置く。

## Alternatives considered

- **コレオグラフィ (中央調整なし)**: 各サービスがイベント連鎖で進む。結合は緩いが、跨ぐステップが増えると「サーガの現在地」が分散し可視性・デバッグが落ちる。order FSM を単一の真実にするオーケストレーションを採る。
- **予約・決済を同期のまま (現行踏襲)**: 即時 409 で UX は良いが、量子の可用性が独立せず主駆動特性に反する。
- **専用オーケストレータサービス**: サーガを独立サービスに切る。本リポジトリ規模では量子を増やすだけで、order FSM が既にサーガ状態を持つ選択と重複する。order をオーケストレータにする。
- **失敗を `cancelled` に集約**: 状態は減るが「なぜ終ったか」が status 単体で読めない。`rejected` を分ける。
- **サーガ状態を別テーブルに分離**: order.status を顧客向けに保てるが状態が 2 系統に割れ整合負担が増える。order.status に集約する。
