# ADR-202608150830: 非同期イベントの配送をマネージドキュー前提にする (DLQ はブローカ機能)

- Status: Accepted
- Date: 2026-08-15
- Supersedes: ADR-[[202606211200]] (broker の選定部分のみ。イベント駆動で配送を起動する判断は有効)
- Relates to: ADR-[[202606300600]] (outbox), ADR-[[202606261702]] (補償イベント)

## Context

- Redis Streams はブローカ側に DLQ・再試行上限・滞留の可視化を持たない。毒メッセージの隔離を
  XPENDING + XCLAIM の 2 段と配送回数判定で自作したが (#106)、ブローカが持っていて当然の機能を
  アプリコードで埋める形になり、維持対象が増えるだけだった。
- 運用先は AWS / GCP のマネージドを想定する。特定ブローカ製品への依存は避けたい。
- 1 つのイベントを複数サービスが購読する (`order.cancelled` を payment / shipping / inventory)。

## Decision

配送を「トピック (fan-out) + 購読者ごとのキュー + キューごとの DLQ」の形に寄せ、隔離と再試行上限を
**ブローカの設定**で賄う。アプリに隔離ロジックを持たせない。

- AWS: SNS → SQS、DLQ は redrive policy (`maxReceiveCount`)。GCP: Pub/Sub のトピック →
  サブスクリプション、dead letter topic。どちらも各クラウドの既定手段を選ぶ。
- ローカル / CI は kumo (MIT・Go 単一バイナリ) を使う。SNS→SQS の配信と redrive policy を実装しており、
  LocalStack より軽い。
- outbox (ADR-[[202606300600]]) は据え置き。発行の保証はブローカ選定と独立している。

## Consequences

- `redisx` の claim / PEL、DLQ 実装、`messaging.dlq.depth` の計装が不要になる。滞留は CloudWatch /
  Cloud Monitoring の標準メトリクスで見る。
- **リプレイを失う**。SQS に保持は無く、Pub/Sub の seek も 7 日。ECST + read model
  (ADR-[[202607020324]]) の再構築は別手段が要る。
- 標準キューは順不同。順序が要る箇所は SQS FIFO / Pub/Sub の ordering key に注文 ID を使う。
- ローカルと本番で実装が異なる (kumo は in-flight を永続化しない)。挙動差を踏む可能性が残る。
- Redis はこの用途では不要になる。

## Alternatives considered

- **Kafka (MSK / GCP Managed Service for Apache Kafka)** — リプレイ・保持・パーティション順序は最良だが、
  DLQ はアプリ側 (retry topic + DLQ topic) のままで、自作をやめるという本 ADR の目的を満たさない。
- **RabbitMQ** — DLX が標準で使い勝手は最良。GCP に一次マネージドが無く、製品への依存も残る。
- **NATS JetStream** — 軽量で `max_deliver` もあるが、両クラウドともマネージドが無く自前運用になる。
- **Redis Streams 継続** — 自作した隔離ロジックの維持コストと、リプレイ不可がそのまま残る。
