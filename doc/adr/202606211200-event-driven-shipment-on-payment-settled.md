# ADR-202606211200: 決済確定イベントを起点に shipping が配送を手配する (Redis Streams)

- Status: Accepted
- Date: 2026-06-21
- Relates to: ADR-[[202606190900]] (配送を同期 checkout に含めない判断), ADR-[[202606170909]] (顧客系/運用系 DB・network 分離), ADR-[[202606170901]] (codegen-first), ADR-[[202606180903]] (PUT セマンティクス)

## Context

ADR-[[202606190900]] は配送手配を checkout の同期パスから分離し、起動方式 (イベント駆動 / 別エンドポイント) は後続に切り出していた。本 ADR がその起動方式を確定する。論点:

- **トリガは決済確定**であって注文確定ではない。`pending` の直後に枠を切ると未入金出荷になりうる (ADR-[[202606190900]])。
- payment (顧客系) と shipping (運用系) は直接到達できず (ADR-[[202606170909]])、何らかの経路が要る。
- 配送側の不調で決済 API を失敗にしたくない (可用性を巻き込まない)。
- Step 0 にはイベント基盤が無い。

## Decision

決済確定をドメインイベントとして broker (Redis Streams) に publish し、shipping が consumer group で購読して配送枠を作る非同期方式を採る。

- **publish**: payment が確定遷移 (`paid`/`settled`/`captured`) 時に `payment.settled` を stream `payment.events` へ XADD。**subscribe**: shipping が consumer group で XREADGROUP し、当該注文の枠を作り XACK。
- **precondition を持たない**のが決め手: 「決済確定が先」はイベントの発火条件に内包される (確定時のみ publish) ため、shipping 側で payment を同期照会する結合が要らない。
- **冪等性**: at-least-once 前提で `shipments.order_id` を UNIQUE + `ON CONFLICT DO NOTHING` とし「1 注文 1 配送」を強制。重複は no-op で ack。
- 生成コードは引き続き codegen (ADR-[[202606170901]])。

### network 上の broker 配置

broker は payment・shipping 双方から到達するため両 private network に足を持つ。ADR-[[202606170909]] は業務サービスを multi-home させない方針だが、broker は DB と同類の共有 infra で業務サービスではない。edge-proxy が同期の境界越えを担うのと対に、broker が顧客系→運用系の非同期チャネルを担う。

### consumer は別バイナリ (worker) として分離する

shipping を `cmd/server` (HTTP) と `cmd/worker` (consumer) の 2 バイナリ・別デプロイに分ける。決め手:

- HTTP と consumer は負荷も停止条件も別で、**独立にスケール・再起動**したい。
- **fail loud**: HTTP 同居の goroutine だと consumer が静かに死んでも `/healthz` は 200 のままで silent partial outage に気付けない。worker は確定エラーで非ゼロ終了し再起動に委ねる。
- worker を持つのは shipping だけなので、`cmd/` 分割もこのサービスに限り無駄な構成差を増やさない。

### shipping schema の見直し

carrier / tracking_no は手配時点では未確定 (運送会社割当後に埋まる) で、確定時の不変スナップショットではない。作成時必須は誤りだったので緩め、`orderId` だけで preparing 枠を作れるようにした。

## Consequences

- **疎結合・可用性**: payment と shipping が時間的に分離され、配送側ダウン中の確定も復旧後に consumer が拾える。
- **結果整合の宿題 (publish 側)**: 本 Step は outbox 無し。DB commit と XADD は別操作で、commit 後の broker 障害でイベントを取りこぼす。失敗は `slog` で可視化し決済 API は 200 を返す (決済は確定済み)。outbox / 突合・再送は将来 Step。
- **carrier/tracking_no の事後更新は follow-up**: preparing 枠を埋める更新 API と未割当の表現 (空文字 vs null) は対象外。現状は空文字で「未割当」を表す。
- **新規 infra**: compose に broker (redis) が増え、監視対象が 1 つ増える。
- **DLQ なし**: 恒久的に失敗するイベントは pending に残る。毒メッセージ隔離は将来課題。

## Alternatives considered

- **別エンドポイントで同期に shipping を叩く**: 基盤不要で素直だが、ADR-[[202606190900]] が避けた「顧客系が運用系を同期駆動する結合」を別口で復活させ、配送側の不調を呼び出し元へ波及させる。不採用。
- **Outbox + relay (Postgres)**: 取りこぼしを無くせる正攻法だが、relay・ポーリング・境界越え配信の自前実装が Step 0 に過大。まず broker で疎結合を作り、信頼配信は後で足す。
- **Redis Pub/Sub**: at-most-once・非永続で、購読者不在時にメッセージが消える。shipping 再起動中の確定が「入金済みなのに配送が作られない注文」になるため不採用。本 ADR の可用性・冪等化は永続ログ + consumer group + ack/再配送を持つ Streams で初めて成立する。Pub/Sub は取りこぼし許容の fan-out 向き。
- **分散トランザクション / saga + 補償**: 下流まで強整合にできるが Step 0 には過剰 (ADR-[[202606190900]] と同じ理由)。
