# ADR-202606300600: Outbox を専用テーブル + 共有ディスパッチャに作り替える

- Status: Accepted
- Date: 2026-06-30
- Supersedes: ADR-[[202606261212]]
- Relates to: ADR-[[202606261214]] (consumer 冪等性), ADR-[[202606250159]] (span link 伝播), ADR-[[202606261702]] (order.cancelled 補償), ADR-[[202606240522]] (DB-per-domain), ADR-[[202606261216]] (致命/縮退の分類)

## Context

- ADR-[[202606261212]] は集約テーブルに送信状態列を足す方式で、「イベント 1 種なら集約列で足り、増えたら専用テーブルへ」と移行トリガを明記していた。#88 で 2 種目 (order.cancelled) が増え、集約に発行プラミング列が累積し、1 集約 = 1 イベント種に縛られ汎用 payload を持てない。移行トリガに達した。
- 受信側は既に pub/sub（Redis Streams + consumer group）。論点はバスの選択ではなく、production 側で「確実にバスへ載せる」能力を発行種が増えても破綻させないこと。

## Decision

1. **専用テーブル**: サービスごとに専用 `<schema>.outbox`（汎用 `payload jsonb` を持つ）を切り、集約から `*_event_*` 列を落とす。業務更新と outbox INSERT を同一トランザクションで確定し、リレーが未送信 (`published_at IS NULL`) を polling して `XAdd` する点は不変。
2. **共有ディスパッチャ**: producer はドメインイベントを `outbox.Dispatch` に渡すだけにし、payload 符号化 (`Dispatch`) と復号 (`DecodePayload`) を `server/internal/outbox` の対で単一情報源にする（traceparent 付与も同所）。各サービスは sqlc 生成型を `outbox.Inserter` へ適合させる薄い adapter を 1 つ持つ。
3. **適用範囲**: オンランプ (outbox) を通すのは欠落が損害になるイベントに限る。損失許容イベント（監査・分析等）は直 publish を許す。現状の `payment.settled` / `order.cancelled` は前者。

## Consequences

- 集約から発行の都合の列が消え、発行種が増えても集約スキーマは不変。新 producer は Event を実装して `Dispatch` を呼ぶだけで「commit 済みは必ず送出 (at-least-once)」を継承し、enqueue 忘れと手書きの marshal/INSERT が構造的に消える。
- 発行保証と consumer 冪等性は不変。
- 運用していないため移行時の in-flight 行はバックフィルしない（新規 DB 前提、ADR-[[202606261212]] 同様）。
- リレーは単一インスタンス前提のまま。複数化は `SELECT … FOR UPDATE SKIP LOCKED`（将来）。高スループット / Kafka なら CDC（Debezium Outbox Router）でリレーを外部化する余地。

## Alternatives considered

- **集約列を維持** (ADR-[[202606261212]]): 移行トリガ到達で不採用。
- **各 command で発行を手書き**: producer ごとに重複し enqueue 忘れを誘発。発行増を見越し `Dispatch` に集約。
- **フル Unit-of-Work**（集約が event を raise し tx ラッパが自動 flush）: 書き忘れを型で封じるが、anemic な sqlc 行に domain-event 機構を被せる重さ。現状 2 producer には過剰で、明示呼び出しの `Dispatch` を採る。
- **共有 outbox を 1 テーブルに集約**: DB-per-domain (ADR-[[202606240522]]) を再結合し schema 所有権を崩す。サービスごとに切る。
- **直 publish 全面化 / CDC**: 前者は重要イベントで dual-write 欠落、後者は基盤が重くローカル完結・費用ゼロに反する。損失許容イベントの直 publish と、Kafka 採用時の CDC は将来の選択肢として残す。
