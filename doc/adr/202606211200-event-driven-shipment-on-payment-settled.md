# ADR-202606211200: 決済確定イベントを起点に shipping が配送を手配する (Redis Streams)

- Status: Accepted
- Date: 2026-06-21
- Relates to: ADR-[[202606190900]] (配送を同期 checkout に含めない判断), ADR-[[202606170909]] (顧客系/運用系 DB・network 分離), ADR-[[202606170901]] (codegen-first), ADR-[[202606180903]] (PUT セマンティクス)

## Context

ADR-[[202606190900]] は配送 (shipment) の作成を checkout の同期パスから分離し、「決済確定後に
運用系で手配する」とだけ決めて、起動方式 (イベント駆動 / 別エンドポイント) は後続イシューに
切り出していた。本 ADR はその起動方式を確定する。

制約と論点:

- **トリガは決済確定**。注文確定 (checkout) ではない。`pending` 決済の直後に出荷枠を切ると
  「未入金出荷」になりうる (ADR-[[202606190900]])。
- **境界越え**。payment は顧客系 (external-private)、shipping は運用系 (internal-private) に居て、
  両者は直接到達できない (ADR-[[202606170909]])。何らかの経路が要る。
- **可用性を巻き込まない**。決済が成功しているのに配送側の不調で決済 API を失敗にしたくない。
- Step 0 にはイベント基盤が無い。

## Decision

決済確定を**ドメインイベントとして broker (Redis Streams) に publish** し、shipping が
**consumer group で購読して配送枠を作る**非同期方式を採る。

- **publish (payment)**: `PUT /payments/{id}` で status が確定状態 (`paid` / `settled` /
  `captured`) に遷移したとき、payment が stream `payment.events` に `payment.settled`
  (`orderId` / `paymentId` / `amountCents`) を XADD する。
- **subscribe (shipping)**: shipping が consumer group `shipping` で `payment.events` を
  XREADGROUP し、`payment.settled` を受けて当該 `orderId` の配送枠を作る。処理成功で XACK。
- **precondition は持たない**。「決済確定が先」はイベントの発火条件そのものに内包される
  (確定遷移したときだけ publish される) ので、shipping 側で payment を同期照会して確認する
  必要がない。
- **冪等性**: at-least-once 配送を前提に、`shipping.shipments.order_id` を UNIQUE にし
  `ON CONFLICT DO NOTHING` で「1 注文 1 配送」を強制する。重複イベントは既存枠に当たって
  no-op になり、consumer は ack して捨てる。
- 生成コードは引き続き codegen (ADR-[[202606170901]])。broker クライアントは go-redis。

### network 上の broker 配置

broker は payment (external-private) と shipping (internal-private) の双方から到達する必要が
あり、両 private network に足を持つ。ADR-[[202606170909]] は「両 network に足を持つのは
edge-proxy / backoffice の 2 proxy だけ」としたが、これは**業務サービス**に対する制約であり、
DB と同類の**共有 infra** である broker は対象外とする。broker は顧客系→運用系を結ぶ
**唯一の非同期チャネル**で、edge-proxy が同期の境界越えを一手に引き受けるのと対になる
(payment は publish のみ・shipping は consume のみ、と用途も絞られる)。

### consumer は別バイナリ (worker) として分離する

consumer は shipping の HTTP プロセスに同居させず、同一コードベースの**別バイナリ・別デプロイ**
にする (web/worker 分離)。shipping は `cmd/server` (HTTP) と `cmd/worker` (consumer) の 2 main を
持ち、compose も `shipping` と `shipping-worker` の 2 サービスに分ける。決め手:

- **独立にスケール・再起動できる**。HTTP と consumer は負荷も停止条件も別。
- **fail loud**。worker は Run が確定エラーを返したら非ゼロ終了し、オーケストレータの再起動に
  委ねる。HTTP に同居させた goroutine だと consumer が静かに死んでも `/healthz` は 200 のままで
  気付けない (silent partial outage)。
- worker を持つのは shipping だけなので、`cmd/` 分割もこのサービスに限る。他サービスは単一 main の
  ままにし、無駄な構成差を増やさない。

### shipping schema の見直し

carrier / tracking_no は配送手配の時点では未確定で、運送会社割当後に埋まる。確定時の業務
事実 (商品名・単価のような不変スナップショット) ではないので、作成時必須は設計の誤り
だった。`NOT NULL DEFAULT ''` に緩め、status は `DEFAULT 'preparing'` にして、`orderId` だけで
preparing 枠を作れるようにした。

## Consequences

- **疎結合・可用性**: payment と shipping が時間的に分離される。配送側が落ちていても決済
  確定は成立し、復旧後に consumer が pending を引き取って手配できる。
- **結果整合の宿題 (publish 側)**: 本 Step は outbox を持たない。決済確定の DB commit と
  XADD は別操作で、commit 後に broker 障害が起きるとイベントを取りこぼす。失敗は `slog` で
  可視化し決済 API 自体は 200 を返す (決済は確定済みのため)。トランザクショナル outbox /
  突合・再送は将来 Step に回す。
- **冪等性は consumer 必須**: at-least-once なので、重複・再配送が前提。UNIQUE(order_id) +
  ON CONFLICT で吸収する。
- **carrier/tracking_no の事後更新は follow-up**: preparing 枠を後から埋める更新 API
  (`PUT /shipments/{id}` の carrier/tracking_no 対応) と、API 応答での未割当の表現 (空文字 vs
  null) は本 PR の対象外。現状は status のみ更新でき、carrier/tracking_no は空文字で「未割当」を表す。
- **新規 infra**: compose に broker (redis) が増える。運用・監視対象が 1 つ増える。
- **DLQ なし**: handle が恒久的に失敗するイベントは pending に残り続ける。毒メッセージの隔離
  (DLQ) は未対応で将来課題。

## Alternatives considered

- **(別エンドポイント) order/backoffice が同期で shipping を叩く**: イベント基盤が要らず素直
  だが、顧客系が運用系を同期駆動する結合 (ADR-[[202606190900]] が避けた形) を別口で復活させ、
  配送側の不調を呼び出し元に波及させる。非同期の疎結合を選んで不採用。
- **Outbox + relay (Postgres ベース)**: publish 取りこぼしを無くせる正攻法だが、relay
  プロセス・ポーリング・境界越えの配信を自前で組む実装量が Step 0 に対して過大。まず broker で
  疎結合の形を作り、信頼配信 (outbox) は後で足す。
- **Redis Pub/Sub (PUBLISH/SUBSCRIBE)**: 同じ Redis でもこちらは at-most-once・非永続で、
  publish 時に購読者が居なければメッセージが消える。shipping の再起動中に決済が確定すると
  「入金済みなのに配送が永遠に作られない注文」が出るため不採用。本 ADR の可用性の主張 (配送側
  ダウン中の確定を復旧後に拾う) と冪等化 (at-least-once 前提) は、永続ログ + consumer group +
  ack/再配送を持つ Streams だからこそ成り立つ。Pub/Sub が向くのは取りこぼし許容の fan-out
  (ライブ通知・キャッシュ無効化ヒント等) で、業務イベントには合わない。
- **分散トランザクション / saga + 補償**: 下流まで強整合にできるが Step 0 には過剰
  (ADR-[[202606190900]] と同じ理由)。
