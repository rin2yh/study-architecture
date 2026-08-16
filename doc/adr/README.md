# ADR 一覧

設計判断の記録。ID はファイル名先頭の作成日時タイムスタンプ (`YYYYMMDDHHmm`)。採番規約は
[ADR-202606211000](202606211000-adr-timestamp-naming.md)。相互参照は本文中で `ADR-[[<ID>]]` 形式で張る。

領域ごとに分けて並べる (領域内は ID 昇順)。全体の入口は
[ADR-202607011621](202607011621-microservices-migration-target-and-drivers.md) で、主駆動特性と移行原則を
そこで宣言し、他の判断はおおむねそれへの奉仕として説明している。

## スタイル・境界

量子の粒度と分割の進め方。

| ID | Status | タイトル |
| --- | --- | --- |
| [202606170900](202606170900-service-based-architecture.md) | Accepted (target style は ADR-[[202607011621]] で移行中) | サービスベースアーキテクチャを採用する |
| [202606170909](202606170909-split-customer-and-ops-db.md) | Accepted | 顧客系 (社外) と運用系 (社内) で DB とネットワーク経路を分離 |
| [202606220716](202606220716-ci-split-by-architecture-quantum.md) | Accepted | CI をアーキテクチャ量子 (顧客系 / backoffice) ごとに 2 ワークフローへ分割する |
| [202606240522](202606240522-step3-split-db-per-domain-from-weak-edge.md) | Accepted | Step 3 で結合の弱い縁からドメインごとに DB インスタンスを分割する |
| [202606262000](202606262000-inventory-as-independent-service.md) | Accepted (戻しの表現は一部 ADR-[[202606281000]] で補足) | 在庫を独立サービス (独自 DB・量子) として切り出す |
| [202607011621](202607011621-microservices-migration-target-and-drivers.md) | Accepted | サービスベースからマイクロサービスへ段階移行する (主駆動特性と移行原則) |

## order のライフサイクルとサーガ

移行の背骨。checkout の同期連鎖を解く一連の判断。

| ID | Status | タイトル |
| --- | --- | --- |
| [202607011720](202607011720-order-lifecycle-state-machine.md) | Accepted (語彙は ADR-[[202607020305]] で拡張) | order のライフサイクルを状態機械 + 追記履歴で表す |
| [202607020305](202607020305-order-checkout-orchestration-saga.md) | Accepted | checkout を order オーケストレーション型サーガにし前進フローを非同期化する |

## データ所有と横断参照

誰がどのデータを持ち、他ドメインの必要分をどう得るか。

| ID | Status | タイトル |
| --- | --- | --- |
| [202606170903](202606170903-shared-postgres-schema-per-domain.md) | Accepted | 共有 Postgres + ドメインごとの schema 分離 |
| [202606190900](202606190900-cross-domain-snapshot.md) | Accepted | 横断データは注文確定時に order 側へスナップショット保存する |
| [202606231000](202606231000-enforce-schema-ownership-with-db-roles.md) | Accepted | データ所有権を schema ごとの最小権限 DB ロールで強制する |
| [202606261704](202606261704-shipping-address-book-and-order-snapshot.md) | Accepted | 配送先住所は member の住所帳で持ち、注文時に order/shipment へスナップショットする |
| [202606301000](202606301000-shipping-pulls-destination-from-order.md) | Accepted | 配送先スナップショットは shipping が order から引く (settled は orderId のみ) |
| [202606301100](202606301100-bff-resolves-shipping-address-for-checkout.md) | Accepted | checkout の配送先は BFF が解決し order へ値で渡す (order は member を引かない) |
| [202607020324](202607020324-cross-domain-read-model-ecst.md) | Accepted | 横断参照データの既定を ECST + ローカル read model にする (役割で snapshot / pull と使い分け) |

## 信頼性・トランザクション

イベント発行の保証、冪等、障害時の振る舞い。

| ID | Status | タイトル |
| --- | --- | --- |
| [202606211200](202606211200-event-driven-shipment-on-payment-settled.md) | Accepted (broker の選定は ADR-[[202608150830]] で Superseded) | 決済確定イベントを起点に shipping が配送を手配する (Redis Streams) |
| [202606261210](202606261210-sync-call-resilience-policy.md) | Accepted | order の同期呼び出しに timeout・リトライ・サーキットブレーカを入れる |
| [202606261212](202606261212-transactional-outbox-payment-settled.md) | Superseded | 決済確定イベントを Transactional Outbox (集約列 + プロセス内リレー) で確実に発行する |
| [202606261214](202606261214-idempotency-checkout-and-shipping.md) | Accepted | checkout と shipping を DB ユニーク制約で冪等にする |
| [202606261216](202606261216-graceful-degradation-policy.md) | Accepted | 依存を「致命 / 縮退可」で分類しグレースフルデグラデーション方針を定める |
| [202606261700](202606261700-inventory-two-phase-reservation-ledger.md) | Superseded | 在庫を append-only の在庫変動台帳で持ち、予約→確定の2フェーズで引き当てる |
| [202606261702](202606261702-order-cancel-event-driven-compensation.md) | Accepted | 注文キャンセルの補償をイベント駆動 (order.cancelled) で各サービスに分散する |
| [202606281000](202606281000-inventory-cancel-confirmed-reservation.md) | Accepted | 確定済み予約のキャンセル戻しを予約行の cancelled_at で表す |
| [202606300600](202606300600-transactional-outbox-table-and-dispatcher.md) | Accepted | Outbox を専用テーブル + 共有ディスパッチャに作り替える (ADR-202606261212 を Supersede) |
| [202608122000](202608122000-order-id-value-object-without-numeric-representation.md) | Accepted | 注文 ID を値オブジェクトにし、数値表現をデータ層に閉じる |
| [202608150830](202608150830-managed-queue-with-broker-side-dlq.md) | Accepted | 非同期イベントの配送をマネージドキュー前提にする (DLQ はブローカ機能) |
| [202608150835](202608150835-broker-behind-port.md) | Accepted | ブローカをポートで隔離し、アプリからベンダ API を見せない |

## 可観測性

計装・収集経路・アラート。

| ID | Status | タイトル |
| --- | --- | --- |
| [202606241356](202606241356-observability-otel-collector-grafana.md) | Accepted | 可観測性を OpenTelemetry + Grafana Alloy + Grafana スタックで構築する |
| [202606241420](202606241420-metrics-push-to-collector-pull-by-prometheus.md) | Accepted (一部 Superseded) | メトリクスはアプリから Alloy へ push し Prometheus は Alloy を scrape する |
| [202606250141](202606250141-telemetry-sensitive-data-masking.md) | Accepted | テレメトリの秘匿情報は計装段と Alloy 段の二重でマスキングする |
| [202606250159](202606250159-trace-async-redis-streams-with-span-link.md) | Accepted | Redis Streams の非同期イベントは traceparent + span link でトレースをつなぐ |
| [202606251000](202606251000-metrics-alloy-push-to-prometheus-otlp.md) | Accepted | メトリクスは Alloy から Prometheus へ OTLP で push する |
| [202606261100](202606261100-alerts-grafana-managed-provisioned.md) | Accepted | アラートは Grafana-managed alerting で provisioning する |
| [202606261600](202606261600-resource-metrics-cadvisor-via-alloy.md) | Accepted | リソースメトリクスは cAdvisor を Alloy で収集し既存 OTLP 経路へ相乗りさせる |

## 認証・信頼境界

| ID | Status | タイトル |
| --- | --- | --- |
| [202606211100](202606211100-member-auth-httponly-cookie-session.md) | Accepted | 認証は HttpOnly Cookie + member 所有のサーバ側セッション |
| [202606230930](202606230930-bff-auth-context-and-trust-boundary.md) | Accepted | store BFF に認証コンテキストの集約と X-Member-Id の付与点を置く |

## API 契約

| ID | Status | タイトル |
| --- | --- | --- |
| [202606180901](202606180901-api-error-model.md) | Accepted | API エラーモデルを共通 Error スキーマ + ErrorJSON ミドルウェアに集約する |
| [202606180903](202606180903-update-endpoint-put-semantics.md) | Accepted | 更新エンドポイント (PUT) はドメイン上ミュータブルな属性のみ置換する |

## 技術スタック・コード生成

| ID | Status | タイトル |
| --- | --- | --- |
| [202606170901](202606170901-codegen-first-tech-stack.md) | Accepted | コード生成中心の技術スタック |
| [202606170902](202606170902-single-root-gomod-monorepo.md) | Accepted | 単一ルート go.mod のモノレポ構成 |
| [202606170907](202606170907-go-web-framework-gin.md) | Accepted | Go サーバの HTTP フレームワークに Gin を採用 |
| [202606180900](202606180900-migration-per-service.md) | Accepted | マイグレーションをサービスごとに分割する |
| [202606190901](202606190901-client-generated-code-layout.md) | Accepted | orval 生成物を gen/ ディレクトリに集約する |

## フロントエンド

| ID | Status | タイトル |
| --- | --- | --- |
| [202606170904](202606170904-frontend-tanstack-start.md) | Superseded | フロントエンドは TanStack Start（BFF を見据える） |
| [202606170905](202606170905-ui-server-loader-data-fetching.md) | Accepted | UI のデータ取得はサーバ側ローダ + orval(zod) |
| [202606170906](202606170906-frontend-pnpm-monorepo-tooling.md) | Accepted | フロントエンドは pnpm モノレポ + oxlint/oxfmt、命名は client/server・単数 |
| [202606170908](202606170908-frontend-react-router-v7.md) | Accepted | フロントエンドは React Router v7 (旧 Remix 統合) に切替 |
| [202606220300](202606220300-frontend-fsd-component-layering.md) | Accepted | フロントエンドのコンポーネントを Feature-Sliced Design で層分けする |
| [202606221300](202606221300-merge-mypage-into-store.md) | Accepted | 社外フロントを store に統合し mypage を退役する |

## テスト・開発規律

規約を機械で守る仕組みもここに置く。

| ID | Status | タイトル |
| --- | --- | --- |
| [202606180902](202606180902-repository-real-db-integration-test.md) | Accepted | repository 層は実 DB 結合テストで検証する |
| [202606190902](202606190902-parallel-integration-test-template-db.md) | Accepted | 結合テストはテンプレート DB クローンで分離し並列実行する |
| [202606190903](202606190903-repository-cqrs-query-command.md) | Accepted | repository を CQRS で Query / Command に分割する |
| [202606210900](202606210900-what-comment-lint-via-claude-hook.md) | Accepted | what コメント検出を Claude Code hook + LLM 判定で行う |
| [202606211000](202606211000-adr-timestamp-naming.md) | Accepted | ADR の識別子を連番からタイムスタンプ (YYYYMMDDHHmm) に変える |
| [202606211520](202606211520-test-case-class-4xx-quasi-normal.md) | Accepted | テストのケース分類で 4xx を準正常系・5xx を異常系に分ける |
| [202607020343](202607020343-fitness-functions-in-ci.md) | Accepted | アーキテクチャ特性を CI のフィットネス関数で守る (第一号は量子越え同期呼び出しの許可リスト) |
| [202608150859](202608150859-docs-freshness-auto-pr.md) | Accepted | ドキュメント鮮度を AI の自動 draft PR で維持する |
