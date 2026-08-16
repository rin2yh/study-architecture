# アーキテクチャ

## サービス構成

出発点はサービスベース（[ADR-202606170900](adr/202606170900-service-based-architecture.md)）で、現在は
マイクロサービスへ段階移行中（[ADR-202607011621](adr/202607011621-microservices-migration-target-and-drivers.md)）。

- ドメインサービス 6 つ（product / order / payment / member / shipping / inventory）。各 1 コンテナ・
  個別デプロイ。shipping と inventory は非同期イベントを処理するワーカーを別コンテナで持つ。
- DB はドメインごとに独立インスタンス（[ADR-202606240522](adr/202606240522-step3-split-db-per-domain-from-weak-edge.md)）。
- 顧客系（社外）と運用系（社内）で network を分け、社外からの到達は edge-proxy が中継する
  （[ADR-202606170909](adr/202606170909-split-customer-and-ops-db.md)）。
- 非同期イベントは broker（SNS + SQS 互換のマネージドキュー。
  [ADR-202608150830](adr/202608150830-managed-queue-with-broker-side-dlq.md)）を通す。

コンテナ・ホストポート・profile・DSN の実体は `compose.yaml` を参照する。

## API

各サービスは `GET /healthz`（liveness）に加え、一覧 / 取得 / 作成 / 更新を持つ
（例: `GET`/`POST /products`、`GET`/`PUT /products/{id}`）。エラー整形と 404/409/422 の扱いは
[ADR-202606180901](adr/202606180901-api-error-model.md)、更新 (PUT) の項目方針（業務項目のみ置換・
FK 不変）は [ADR-202606180903](adr/202606180903-update-endpoint-put-semantics.md)。エンドポイントの
一覧は各サービスの `server/<svc>/api/openapi.yaml` が単一情報源。

## 注文の流れ

order の `POST /checkout` がカート（`productId` + `quantity`）を確定する。product を参照して商品名・
単価を注文明細へスナップショットし（[ADR-202606190900](adr/202606190900-cross-domain-snapshot.md)）、
inventory へ在庫を予約し（[ADR-202606262000](adr/202606262000-inventory-as-independent-service.md)）、
payment を手配する。この 3 つはいずれも同期 HTTP。

配送は同期 checkout に含めず、決済確定イベントを起点に shipping が非同期で手配する
（[ADR-202606211200](adr/202606211200-event-driven-shipment-on-payment-settled.md)）。キャンセルの
補償もイベント駆動（[ADR-202606261702](adr/202606261702-order-cancel-event-driven-compensation.md)）。

order の状態は状態機械 + 追記履歴で表す
（[ADR-202607011720](adr/202607011720-order-lifecycle-state-machine.md)）。前進フローをサーガとして
非同期化する決定は [ADR-202607020305](adr/202607020305-order-checkout-orchestration-saga.md) にあるが、
checkout（`server/order/internal/handler/order_write.go`）は現時点では同期のまま。
