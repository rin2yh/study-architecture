# ADR-202606241356: 可観測性を OpenTelemetry + Grafana Alloy + Grafana スタックで構築する

- Status: Accepted
- Date: 2026-06-24
- Relates to: ADR-[[202606170900]] (サービスベース採用), ADR-[[202606240522]] (Step 3 ドメインごと DB 分割), ADR-[[202606211200]] (payment settled → shipping のイベント駆動 / Redis Streams), ADR-[[202606230930]] (store BFF の信頼境界), ADR-[[202606220716]] (CI のデプロイ量子分割), ADR-[[202606241420]] (メトリクスの取り込み方式)

## Context

Step 3 (ADR-[[202606240522]]) で DB をドメインごとに剥がし、1 リクエストが複数サービス・複数
DB・ブローカーをまたぐようになった。現状は各サービスが `slog` (`*Context` 版) で**サービス単位**の
ログを出すだけで、

- 1 つの checkout が order → payment → (settle イベント) → shipping をどう流れたかを 1 本の線で追えない
- 障害時にどのホップで失敗・遅延したかを横断で特定できない
- レイテンシ・エラー率・スループットの定量が無い (CLAUDE.md「推測するな、計測せよ」に対し計測基盤が未整備)

issue #59 (Step 4) でトレーシング (#60) / ログ相関 (#61) / メトリクス (#62) を横断導入する。
本 ADR はその土台となる**スタックと伝播方式**を確定する。これらは全サービス + compose + CI に
波及し、後から覆すコストが高いため、4-1 着手前に決める (ADR 規約の 2 条件を満たす)。

費用ゼロ・ローカル完結 (README 方針) を維持し、OSS で構成する。

## Decision

### スタック: OpenTelemetry SDK → Grafana Alloy → Grafana (Tempo / Loki / Prometheus)

計装は **OpenTelemetry**。収集役（"collector" 役）には **Grafana Alloy** を 1 つ置き、トレース/メトリクス/
ログの 3 信号をすべて Alloy が捌く。backend を丸ごと Grafana スタック (トレース=Tempo、ログ=Loki、
メトリクス=Prometheus、可視化=Grafana) に寄せるため、収集役も Grafana 公式の paved path で揃える。
Alloy は OTel Collector のコンポーネントに Prometheus / Loki 収集を統合した Grafana 公式エージェントで、
アプリ側の計装は OTLP のままなので将来 OTel Collector へ入れ替えてもアプリは無改修。

- トレース/メトリクス: 各サービスが **OTLP/gRPC で Alloy へ送る**。
- ログ: 各サービスは **stdout に JSON を書くだけ**で、Alloy が収集して Loki へ送る (12-factor の王道。
  アプリにログ送信を持たせない)。`trace_id` / `span_id` は stdout の JSON に載せて相関させる (#61)。

4-1 (#60) ではトレースのみ稼働させるため compose に **Alloy + Tempo + Grafana** を立てる。4-2 (#61) で
Loki、4-3 (#62) で Prometheus を足す。**Alloy を必ず経由する**ことで、backend を増やしても各サービスの
向き先 (Alloy 固定) を触らずに済む。

### 4-1 のスコープは Go 5 サービス + shipping-worker。BFF (Node) は 4-4 で別 PR

4-1 は Go 側 (5 サービス + `shipping-worker`) に限定する。store BFF (Node / TanStack Start,
ADR-[[202606230930]]) の計装は OTel エコシステムが別で PR を分けた方が安全なため、Step 4 内の別サブ
イシュー **4-4 (#64)** として 4-1 とは別 PR で実装する。BFF を入れるとトレースの起点がブラウザ入口まで
上がる。edge-proxy (nginx) はリクエストヘッダを upstream へ素通しするので `traceparent` は伝播し、Go
サービス間は自動でつながる。nginx 自身に span は持たせない。

### 共通化と env 規約

- TracerProvider / MeterProvider の初期化と shutdown を `server/internal/otelx` (新規) に集約する。
  `otelx` は**初期化して shutdown 関数を返す**形にし、HTTP サーバ (`httpx.Serve` の shutdown 経路) と
  worker (`cmd/worker` の `run`) の双方から `defer shutdown()` で flush する。worker は `httpx.Serve` を
  通らないため明示的な flush が要る。
- HTTP サーバ計装は `httpx.NewEngine()` に contrib の `otelgin` を 1 行足し、5 サービス一括で計装する。
  span 名はルートテンプレート (`/orders/:id`) を使い、生パスによるカーディナリティ爆発を避ける。
- サービス間 HTTP は `server/internal/httpx` に `otelhttp.NewTransport` でラップした共有 `http.Client` を
  1 つ置き、order の gateway (product / payment 生成クライアントの `WithHTTPClient`) と後続の BFF 経路で
  共用する。`traceparent` 注入は transport が ctx から行う。
- exporter は **OTLP/gRPC**。エンドポイント・サービス名は標準の `OTEL_*` 環境変数
  (`OTEL_EXPORTER_OTLP_ENDPOINT` / `OTEL_SERVICE_NAME`) で注入し、SDK の自動読み取りに任せて
  `otelx` の初期化を薄く保つ。
- サンプリングは学習用途のため `always_on`。
- **Alloy 不在でもアプリは起動・応答し続ける**: OTLP exporter の接続失敗は致命にしない (graceful
  degradation)。e2e / CI で可観測性スタックを立てない前提のため必須。

### 非同期 (Redis Streams) はコンテキストを wire 契約に載せ span link でつなぐ

go-redis の OTel 計装は redis コマンドを span 化するだけで、stream メッセージをまたいだ trace 伝播は
しない (producer=payment と consumer=shipping は別プロセス・別時刻)。そこで `paymentevent` の wire 契約
(ADR-[[202606211200]]) に **`traceparent` フィールドを足し**、producer が inject・consumer が extract する。
consumer 側は**親子ではなく span link** で結ぶ。決済確定の発行トレースと配送手配の消費トレースは別の
処理単位であり、親子にすると配送の遅延が checkout のレイテンシに混ざるため。

## Consequences

- **1 リクエストを 1 トレースで追える**: ブラウザ手前 (BFF) を除き、edge-proxy 越しの Go サービス間と
  Redis Streams 越しの非同期手配まで 1 本 (+link) で可視化できる。
- **Alloy 経由のコスト**: compose の起動対象に Alloy + Tempo + Grafana が増える。各 Step でバックエンドを
  足す手間も発生する。ただし e2e / CI (ADR-[[202606220716]] の 2 ワークフロー) では可観測性スタックを
  立てない (下記)。
- **向き先の安定**: アプリの送り先は Alloy 固定なので、Tempo→他バックエンド差し替えや Loki/Prometheus
  追加で各サービスの env を触らずに済む。
- **テレメトリは e2e で assert しない**: span/metric が出ることをテストで gate せず、手動 verify と
  上記の graceful degradation で担保する。e2e (`scripts/e2e-up.sh`) に Alloy 等は足さない。
- **wire 契約の拡張**: `paymentevent` に `traceparent` フィールドが増える。producer/consumer 双方の
  改修が要るが、契約は単一ファイルに閉じているため波及は局所。
- **Grafana を最初から**: Jaeger all-in-one より初期構成は重いが、4-2 ログ・4-3 メトリクスで UI を
  一本化でき、4-1 後に backend を載せ替える手戻りが無い。
- **BFF は線が途中から**: 4-1 時点でトレースは最初に当たる Go サービス起点になり、ブラウザ→BFF の
  最初のホップは欠ける。4-4 (#64) で解消する。

## Alternatives considered

- **Jaeger all-in-one で 4-1 を出し、後で Grafana へ移行**: トレースのみなら軽く最短だが、4-2/4-3 で
  ログ・メトリクスを足す段で UI とバックエンドを作り直す手戻りが出る。最初から Grafana に寄せる。
- **収集役を挟まずアプリから backend へ直結**: ホップが 1 つ減るが、backend 追加・差し替えのたびに
  全サービスの送り先を変える必要があり、横断変更が頻発する。Alloy を緩衝にする。
- **収集役を標準 OTel Collector にする / Collector + Alloy 併用**: Collector は CNCF 標準で中立だが、backend を
  丸ごと Grafana に寄せた以上は公式 paved path の Alloy 1 つに揃える方が噛み合う。併用は機能的に問題ない
  が部品が 2 つになり学習用には過剰。
- **Redis Streams を親子 span で継続**: 実装は単純だが、非同期手配の所要時間が発行側トレースの
  span に積算され「checkout が遅い」と誤読させる。span link で別トレースとして関連だけ残す。
- **BFF (Node) も 4-1 で一括計装**: 最初のホップまで線がつながるが、Go と Node で OTel の入れ方・
  依存が分かれ 1 PR が肥大化する。スコープを Go に絞り別 PR に分ける。
