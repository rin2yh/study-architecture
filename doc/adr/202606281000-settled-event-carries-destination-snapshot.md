# ADR-202606281000: 配送先スナップショットは settled イベントに載せ payment が中継する

- Status: Accepted
- Date: 2026-06-28
- Relates to: ADR-[[202606261704]] (この実装で詰める点), ADR-[[202606261212]] (Transactional Outbox), ADR-[[202606250159]] (traceparent 中継), GitHub #89

## Context

- ADR-[[202606261704]] は「shipment は settled イベント経由で宛先スナップショットを受け取る」と決めたが、「shipping が order を引く経路にするか」は実装で詰めるとして残した。
- settled イベントは payment のアウトボックス (payments 行の列) から発行される (ADR-[[202606261212]])。宛先は order が確定するため、shipment へ届けるには経路の選択が要る。

## Decision

宛先スナップショットを settled イベント payload に載せ、payment は解釈せず中継する。

- order は checkout で member から引いた宛先を payment 作成時に同梱する。payment は宛先を専用の中継列に保持し、settled イベントに載せて送出する (traceparent と同じ中継のみ。ADR-[[202606250159]])。
- shipping は consumer がイベントから宛先を読み shipment に保持する。
- 決め手: shipping→order の同期依存を作らず、shipment 手配を settled イベントだけで完結させる (疎結合)。

## Consequences

- shipping は order を同期で引かない。settled イベントが宛先を自己完結で運ぶ。
- payment が配送関心の宛先列を持つ (非対称な持ち分)。payment は中身を解釈しない中継に留め、traceparent 中継と同型に扱うことで漏れを限定する。
- settled イベントと payment スキーマに宛先フィールドが増える (paymentevent.Settled / payment.payments)。

## Alternatives considered

- **shipping が settled 受信時に order を引く**: payment は宛先を持たず純粋だが、shipping→order の同期依存と実行時結合が増える。疎結合の方針に反する。
