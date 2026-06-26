# ADR-202606261214: checkout と shipping を DB ユニーク制約で冪等にする

- Status: Accepted
- Date: 2026-06-26
- Relates to: ADR-[[202606261210]] (POST リトライ解禁の前提), ADR-[[202606261212]] (outbox の at-least-once を吸収), ADR-[[202606211200]] (決済確定イベントで配送手配)

## Context

- 5-1 の同期リトライ (ADR-[[202606261210]]) と 5-2 の at-least-once 配信 (ADR-[[202606261212]]) により、同一 checkout / 同一決済確定イベントが複数回処理され得る。冪等性が無いと決済・配送が重複する。

## Decision

checkout (order→payment) と shipping consumer を冪等にする。

- **idempotency key は order が checkout 受付時に発番**し payment へ渡す。サーバ側に閉じ BFF/UI を改修しない。order→payment のリトライ (ADR-[[202606261210]]) と直接かみ合う。
- **重複検知は DB ユニーク制約で強制する**。payment は idempotency key にユニーク制約を張り二重 INSERT を原子的に弾く。shipping consumer は処理済みイベント (イベントID/決済ID) にユニーク制約を張り再配信を弾く。アプリ側のロックや check-then-act を持たず、競合は DB が保証。
- 適用範囲は checkout の決済作成と shipping の配送手配。member セッション等は範囲外。

## Consequences

- order→payment POST のリトライ (ADR-[[202606261210]]) を安全に解禁できる。
- 5-2 の at-least-once 重複が shipping 側で吸収され、配送が重複しない。
- payment / shipping に migration (キー列 + ユニーク制約) が要る。
- 制約違反は「既に処理済み = 成功扱い」へ変換する (エラーで落とさない。[[error-handling.md]])。

## Alternatives considered

- **クライアント (ブラウザ) 発番**: ユーザーのリロード連打まで吸収できるが BFF/UI 改修でスコープが広がる。サーバ起点の二重 (リトライ/再配信) は order 発番で足りる。
- **consumer 側の処理済みマーク (アプリチェック)**: check-then-act の競合を自前で払う必要があり、DB ユニーク制約なら原子性を DB に委ねられる。
