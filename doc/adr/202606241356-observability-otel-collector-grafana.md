# ADR-202606241356: 可観測性を OpenTelemetry + Grafana Alloy + Grafana スタックで構築する

- Status: Accepted
- Date: 2026-06-24
- Relates to: ADR-[[202606240522]] (Step 3 DB 分割), ADR-[[202606241420]] (メトリクス取り込み), ADR-[[202606250159]] (Redis 非同期伝播), ADR-[[202606250141]] (マスキング)

## Context

Step 3 (ADR-[[202606240522]]) で DB をドメインごとに分け、1 リクエストが複数サービス・DB・ブローカーを
またぐようになった。横断のトレース・ログ相関・メトリクスが無く、どのホップで遅延/失敗したか追えない。
費用ゼロ・ローカル完結 (README 方針) で計測基盤を入れる。

## Decision

**OpenTelemetry で計装し、収集役は Grafana Alloy 1 つ、backend は Grafana スタック (Tempo/Loki/
Prometheus)** にする。backend が全部 Grafana なので収集役も公式の Alloy に寄せる。

- トレース/メトリクス: 各サービスが OTLP/gRPC で Alloy へ送る。
- ログ: stdout の JSON を Alloy が収集して Loki へ (アプリにログ送信を持たせない)。`trace_id` で相関 (#61)。
- env は標準 `OTEL_*`、サンプリングは `always_on`。
- Alloy 不在でもアプリは起動する (exporter 失敗を致命にしない)。e2e/CI に可観測性スタックは足さず、
  テレメトリは assert しない。
- 計装対象は Go 5 サービス + `shipping-worker` (4-1)。BFF は 4-4 (#64) で別 PR。
- アプリ計装は OTLP のままなので、将来 OTel Collector へ入れ替えてもアプリは無改修。

非同期 (Redis) 伝播は ADR-[[202606250159]]、メトリクス取り込みは ADR-[[202606241420]]、マスキングは
ADR-[[202606250141]] に分割。

## Consequences

- 1 リクエストを 1 トレースで追える (4-1 時点は BFF 区間を除く)。
- compose に Alloy + Tempo + Grafana が増え、各 Step で backend を足す。
- 送り先は Alloy 固定なので、backend 追加・差し替えで各サービスを触らない。

## Alternatives considered

- Jaeger all-in-one で出して後で Grafana へ移行 → 4-2/4-3 で作り直し。最初から Grafana。
- 収集役を直結 / 標準 OTel Collector / Collector+Alloy 併用 → backend が全部 Grafana なら Alloy 1 つが
  噛み合う。併用は部品が増え学習用に過剰。
