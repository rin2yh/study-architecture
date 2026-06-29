# ADR-202606281000: 配送先スナップショットは shipping が order から引く (settled イベントは orderId のみ)

- Status: Accepted
- Date: 2026-06-28
- Relates to: ADR-[[202606261704]] (この実装で詰める点), ADR-[[202606250141]] (秘匿情報を必要な所だけに留める), ADR-[[202606170909]] (顧客系/運用系の境界), GitHub #89

## Context

- ADR-[[202606261704]] は「shipment は settled イベント経由で宛先スナップショットを受け取る」と決めたが、宛先の伝搬経路は実装で詰めるとして残した。
- settled イベントは payment のアウトボックスから発行される (ADR-[[202606261212]])。宛先 (宛名・住所) は PII であり、payload に載せると payment 行と Redis ストリーム (`payment.events`) に PII が残る。宛先を必要としない 2 か所に PII を広げることになる。

## Decision

宛先は order から引く。settled イベントには載せない。

- **settled イベントは orderId のみを運ぶ**。payment は宛先を保持も中継もしない。
- **shipping consumer が settled (orderId) を受けて order の `GET /orders/{id}` から宛先スナップショットを引き、shipment に保持する**。order は注文時に宛先を確定した唯一の所有者 (ADR-[[202606261704]])。
- 運用系 shipping → 顧客系 order の越境は edge-proxy 経由とする (backoffice と同じ統制経路。ADR-[[202606170909]])。
- 決め手: PII を必要としない payment と Redis ストリームに残さない (ADR-[[202606250141]] の「秘匿情報は必要な所だけ」)。

## Consequences

- payment は配送関心を持たず、`payment.events` ストリームに PII が乗らない。
- shipment 手配時に shipping→order の取得が 1 回増える。これは consumer 消費時 (バックグラウンド) で checkout 経路ではない。order 不調時は ack せず再配送に委ねる。
- shipping に order 向け生成クライアントと gateway が増える。

## Alternatives considered

- **settled イベントに宛先を載せ payment が中継する** (本 ADR 初稿): shipping→order の同期依存を作らずイベントだけで完結できるが、宛先 (PII) が `payment.payments` と Redis ストリームに残る。秘匿情報を必要な所だけに留める方針 (ADR-[[202606250141]]) と緊張するため差し替えた。consumer 消費時の取得 1 回は、PII の拡散を避ける対価として許容する。
