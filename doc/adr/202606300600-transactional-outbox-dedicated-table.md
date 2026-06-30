# ADR-202606300600: Outbox を集約列から専用 outbox テーブルへ移す

- Status: Accepted
- Date: 2026-06-30
- Supersedes: ADR-[[202606261212]]
- Relates to: ADR-[[202606250159]] (span link 伝播), ADR-[[202606261214]] (consumer 冪等性), ADR-[[202606261702]] (order.cancelled 補償), ADR-[[202606240522]] (DB-per-domain)

## Context

- ADR-[[202606261212]] は「イベント 1 種なら集約列で足り、増えたら専用テーブルへ移行」と移行トリガを明記していた。
- #88 で 2 つ目の発行サービス (order.cancelled) が増え、`order.orders` にも `cancelled_event_*` 3 列が乗った。ドメイン集約に発行プラミング列が累積し、集約列方式では 1 集約 = 1 イベント種に縛られ汎用 payload を持てない。移行トリガに達した。

## Decision

サービスごとに専用 `<schema>.outbox` テーブル (`aggregate_id`, `event_type`, `payload jsonb`, `traceparent`, `published_at`) を切り、集約から `*_event_*` 列を落とす。order / payment を同時に移す。

- 集約更新と outbox INSERT を同一トランザクションで確定する (ローカル整合)。リレーが未送信行 (`published_at IS NULL`) を polling して `XAdd` する点は不変。
- `payload` は発行時の `Values()` をそのまま JSON 化して保存し、復元は `server/internal/outbox.DecodePayload` に共有で寄せる。リレーはイベントの中身を知らないまま複数イベント種を運べる。
- 本リポジトリは運用していないため、移行時の in-flight 未送信行のバックフィルは行わない (列追加 ADR-[[202606261212]] 同様、新規 DB 前提)。

## Consequences

- ドメイン集約から発行の都合の列が消え、イベント種が増えても集約スキーマは不変。1 サービスで複数イベント種を発行できる。
- 発行保証 (commit 済みは再起動後も at-least-once で送出) と consumer 冪等性は不変。
- `payload jsonb` を経由するため bigint は復元時に json.Number から int64 へ戻す (桁落ち回避)。
- リレーは単一インスタンス前提のまま (複数化時の排他は ADR-[[202606261212]] と同じく将来課題)。

## Alternatives considered

- **集約列を維持**: ADR-[[202606261212]] の方式。移行トリガ (2 種目の発行) に達したため不採用。
- **共有 outbox テーブルを 1 つに集約**: DB-per-domain (ADR-[[202606240522]]) を再結合し schema 所有権 (ADR-[[202606231000]]) を崩す。サービスごとに切る。
- **stream をテーブルに保持**: 各サービスは単一 stream へ発行するため、stream は発行側の定数で足り列にしない。
