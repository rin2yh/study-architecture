# ADR-202606300900: イベント発行を共有トランザクショナル・ディスパッチャに集約する

- Status: Accepted
- Date: 2026-06-30
- Relates to: ADR-[[202606300600]] (専用 outbox テーブル), ADR-[[202606261212]] (Outbox 起源), ADR-[[202606261214]] (consumer 冪等性), ADR-[[202606250159]] (span link 伝播), ADR-[[202606261216]] (致命/縮退の分類)

## Context

- ADR-[[202606300600]] で発行を専用 outbox テーブルへ移したが、各 command が `json.Marshal(Values())` ＋ `InsertOutbox` を手書きしていた。発行 producer が増えるほど同じプラミングが各所へ重複し、「業務更新と同一 tx で enqueue し忘れる → 無言で欠落」のリスクが producer ごとに増える。
- 受信側は既に pub/sub（Redis Streams + consumer group）。論点はバスの選択ではなく、production 側で「確実にバスへ載せる」能力を発行種が増えても破綻させないこと。

## Decision

- producer は「起きた事実」をドメインイベント（`EventType()` / `AggregateID()` / `Values()` を実装）として `outbox.Dispatch(ctx, inserter, traceparent, events...)` に渡すだけにする。payload 符号化・行組立・traceparent 付与は共有層が担い、業務更新と同一 tx で outbox へ積む。
- jsonb の符号化（`Dispatch`）と復号（`DecodePayload`）を `server/internal/outbox` の対で単一情報源にし、wire 形のドリフトを防ぐ。
- オンランプ（outbox）を通すのは**欠落が損害になるイベントに限る**。損失許容イベント（監査・分析・通知等）は直 publish を許し過剰なコストを避ける。現状の `payment.settled` / `order.cancelled` は前者。

## Consequences

- 新しい producer / 発行種は Event を実装して `Dispatch` を呼ぶだけで「commit 済みは必ず送出」を継承し、手書きの marshal/INSERT と enqueue 忘れが構造的に消える。
- 各サービスは sqlc 生成型を `outbox.Inserter` へ適合させる薄い adapter を 1 つ持つ（生成型を共有層へ晒さないため）。
- リレーは単一インスタンス前提のまま。複数並べる場合は `SELECT … FOR UPDATE SKIP LOCKED` での claim が要る（将来）。
- 高スループット / Kafka を採るなら CDC（Debezium Outbox Router）でリレーを外部化する余地（ADR-[[202606300600]] と同じく基盤コストとの天秤）。

## Alternatives considered

- **各 command で手書き（移行直後の形）**: 抽象ゼロだが producer ごとに重複し enqueue 忘れを誘発。発行が増える前提で不採用。
- **フル Unit-of-Work（集約が event を raise し tx ラッパが自動 flush）**: 書き忘れを型で完全に封じるが、anemic な sqlc 行へ domain-event 機構を被せる重さ。producer 2・1 種ずつの現状には過剰で、明示呼び出しの `Dispatch` を採る。発行種が増えれば再検討。
- **直 publish 全面化（pub/sub のみ）**: 重要イベントで dual-write 欠落。損失許容イベントに限定して併用する。
