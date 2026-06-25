# ADR-202606250159: Redis Streams の非同期イベントは traceparent + span link でトレースをつなぐ

- Status: Accepted
- Date: 2026-06-25
- Relates to: ADR-[[202606241356]] (可観測性スタック), ADR-[[202606211200]] (Redis Streams イベント駆動), ADR-[[202606250141]] (マスキング)

## Context

payment settled → shipping は Redis Streams のイベント駆動 (ADR-[[202606211200]])。producer と consumer
(`shipping-worker`) は別プロセス・別時刻で、go-redis の計装は redis コマンドを span 化するだけで、
メッセージ越しに trace を伝播しない。checkout から配送手配までを 1 トレースとして追うには伝播方式が要る。

## Decision

`paymentevent` の wire 契約に **`traceparent` フィールドを足し**、producer が inject・consumer が extract。
consumer は**親子でなく span link** で結ぶ。親子だと配送の遅延が checkout のレイテンシ span に積算され
誤読するため。

## Consequences

- 発行と消費は別トレース + link でたどる。「配送が遅い」と「checkout が遅い」を取り違えない。
- `paymentevent` に `traceparent` が増える (単一ファイルで波及は局所)。
- 伝播フィールドは `traceparent` のみ。秘匿情報を混ぜない (ADR-[[202606250141]])。

## Alternatives considered

- 親子 span で継続 → 配送時間が発行側 span に積算され誤読。
