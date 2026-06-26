# ADR-202606261212: 決済確定イベントを Transactional Outbox (集約列 + プロセス内リレー) で確実に発行する

- Status: Accepted
- Date: 2026-06-26
- Relates to: ADR-[[202606211200]] (決済確定イベントで配送手配), ADR-[[202606250159]] (Redis span link 伝播), ADR-[[202606261214]] (冪等性。at-least-once を吸収), ADR-[[202606240522]] (DB-per-domain)

## Context

- payment は決済確定時に「自DB更新」と「Redis Streams へ `XAdd`」の 2 ストアへ書く dual-write (`server/payment/internal/event/event.go`)。コミットと `XAdd` の隙間でプロセスが落ちるとイベントが欠落し、配送が永久に手配されない (ADR-[[202606211200]] の前提が崩れる)。

## Decision

Transactional Outbox でイベント発行を確実にする。

- **専用テーブルを作らず payments 集約に送信状態列 (`published_at` 等) を足す**。「settled への更新」と「未送信状態」を同一トランザクションで確定し、DB とイベントの整合をローカルに閉じる。イベントは決済確定 1 種で汎用 payload が不要なため集約列で足りる。
- **リレーは payment プロセス内の goroutine**。未送信 (settled かつ未 published) 行をポーリングして `XAdd` → `published_at` を埋める。落ちても次回拾い直すので at-least-once で必ず届く。
- リレーロジックは `server/internal/outbox` に共有実装として置く。発行サービスが増えても各サービスが自DBに対し呼ぶだけ。DB-per-domain (ADR-[[202606240522]]) では中央 worker が全DBを再結合してしまうのを避け、in-process が分散・コンテナ増なしで素直。worker へ移すのも共有コードがあるので容易。
- `traceparent` は送信行に保持し (ADR-[[202606250159]] の span link 用)、リレー送出時に inject してトレースを切らさない。

## Consequences

- 決済確定が commit されたイベントは再起動後も必ず Redis へ送出される (欠落しない)。
- 直接 `XAdd` を廃しリレー経由にするため、送出はポーリング間隔ぶん遅延する。
- at-least-once で重複配信し得るため、受信側 (shipping) の冪等性 (ADR-[[202606261214]]) が必須。
- リレーは単一インスタンス前提。payment を複数並べる場合は `SELECT ... FOR UPDATE SKIP LOCKED` 等の排他が要る (現状単一のため将来課題)。

## Alternatives considered

- **専用 outbox テーブル**: 汎用 payload を持てるがイベント 1 種の現状ではオーバー。集約列で足り、増えたら移行。
- **best-effort + 再送**: コミット/`XAdd` 間のクラッシュ欠落を防げず dual-write のまま。
- **CDC (Debezium 等)**: アプリ無改修だが重い基盤が要り、費用ゼロ・ローカル完結に反する。
- **中央リレー worker**: 全ドメインDBへ接続し DB 分離を再結合する。独立スケール/障害分離の要求が出るまで過剰。
