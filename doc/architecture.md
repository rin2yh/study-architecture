# アーキテクチャ

サービス構成と、注文まわりのドメインの振る舞いをまとめる。役割分担: 設計判断は
[ADR](adr/README.md)、プロジェクト概要と環境構築は [README](../README.md)、コード生成と
ディレクトリ構成は [codegen](codegen.md)、UI は [frontend](frontend.md)。

出発点はサービスベース（[ADR-202606170900](adr/202606170900-service-based-architecture.md)）で、
現在はマイクロサービスへ段階移行中（[ADR-202607011621](adr/202607011621-microservices-migration-target-and-drivers.md)）。
同期と非同期が混在する過渡状態にある。

## サービス構成

- ドメインサービス 6 つ（product / order / payment / member / shipping / inventory）。各 1 コンテナ・
  個別デプロイ。shipping と inventory は非同期イベントを処理するワーカーを別コンテナで持つ。
- DB はドメインごとに独立インスタンス（[ADR-202606240522](adr/202606240522-step3-split-db-per-domain-from-weak-edge.md)）。
  各サービスは自身の DB だけを向き、横断 JOIN はしない。schema 所有権は DB ロールで強制する
  （[ADR-202606231000](adr/202606231000-enforce-schema-ownership-with-db-roles.md)）。
- 顧客系（社外）と運用系（社内）で network を分け、社外からの到達は edge-proxy が中継する
  （[ADR-202606170909](adr/202606170909-split-customer-and-ops-db.md)）。中継する path は
  `infra/edge-proxy/nginx.conf`。
- 非同期イベントは broker（SNS + SQS 互換のマネージドキュー。
  [ADR-202608150830](adr/202608150830-managed-queue-with-broker-side-dlq.md)）を通す。送信側は
  Outbox 経由（[ADR-202606300600](adr/202606300600-transactional-outbox-table-and-dispatcher.md)）、
  アプリからは port の裏に隠す（[ADR-202608150835](adr/202608150835-broker-behind-port.md)）。

コンテナ・ホストポート・profile・DSN の実体は `compose.yaml` を参照する（ここには転記しない）。

## API

各サービスは `GET /healthz`（liveness）に加え、一覧 / 取得 / 作成 / 更新を持つ
（例: `GET`/`POST /products`、`GET`/`PUT /products/{id}`）。エラー整形と 404/409/422 の扱いは
[ADR-202606180901](adr/202606180901-api-error-model.md)、更新 (PUT) の項目方針（業務項目のみ置換・
FK 不変）は [ADR-202606180903](adr/202606180903-update-endpoint-put-semantics.md)。

エンドポイントの一覧は各サービスの `server/<svc>/api/openapi.yaml` が単一情報源。

## 注文の流れ

- **checkout**: order の `POST /checkout` がカート（`productId` + `quantity`）を確定する。product を
  参照して商品名・単価を注文明細へスナップショットし（[ADR-202606190900](adr/202606190900-cross-domain-snapshot.md)）、
  inventory へ在庫を予約し、payment を手配する。この 3 つはいずれも同期 HTTP。
- **在庫**: inventory は独立サービスで、在庫を append-only の変動台帳として持ち予約 → 確定の 2 フェーズで
  引き当てる（[ADR-202606262000](adr/202606262000-inventory-as-independent-service.md)）。確定済み予約の
  取り消しは [ADR-202606281000](adr/202606281000-inventory-cancel-confirmed-reservation.md)。
- **配送**: 同期 checkout には含めない。決済確定イベントを起点に shipping が非同期で手配する
  （[ADR-202606211200](adr/202606211200-event-driven-shipment-on-payment-settled.md)）。配送先は
  order から引く（[ADR-202606301000](adr/202606301000-shipping-pulls-destination-from-order.md)）。
  配送先の住所は member の住所帳が権威で、注文時点の値を order へスナップショットする
  （[ADR-202606261704](adr/202606261704-shipping-address-book-and-order-snapshot.md)）。住所 ID を
  値に解決するのは UI 側の BFF（[ADR-202606301100](adr/202606301100-bff-resolves-shipping-address-for-checkout.md)）。
- **キャンセル**: `POST /orders/{id}/cancel`。補償はイベント駆動のコレオグラフィ
  （[ADR-202606261702](adr/202606261702-order-cancel-event-driven-compensation.md)）。
- **状態**: order のライフサイクルは状態機械 + 追記履歴で表す
  （[ADR-202607011720](adr/202607011720-order-lifecycle-state-machine.md)）。前進フロー（予約・決済）を
  サーガとして非同期化する決定は [ADR-202607020305](adr/202607020305-order-checkout-orchestration-saga.md)
  にあるが、`server/order/internal/handler/order_write.go` の checkout は現時点では同期のまま。

## 障害時の振る舞い

- 同期呼び出しの回復性（timeout / retry / circuit breaker）は
  [ADR-202606261210](adr/202606261210-sync-call-resilience-policy.md)。
- 外部依存を「致命 / 縮退可」で分類する方針は
  [ADR-202606261216](adr/202606261216-graceful-degradation-policy.md)。
- 重複配信・再送に対する冪等は [ADR-202606261214](adr/202606261214-idempotency-checkout-and-shipping.md)。

## 認証と信頼境界

会員セッションは httpOnly cookie（[ADR-202606211100](adr/202606211100-member-auth-httponly-cookie-session.md)）。
UI のサーバ側（BFF）が認証コンテキストを解決してサービスへ渡す
（[ADR-202606230930](adr/202606230930-bff-auth-context-and-trust-boundary.md)）。

## 可観測性

トレース / ログ / メトリクスの構成は
[ADR-202606241356](adr/202606241356-observability-otel-collector-grafana.md)。起動方法は
[README](../README.md)、画面の見方は [ops/dashboards](ops/dashboards.md)、アラート発火時の手順は
[ops/runbook](ops/runbook.md)。
