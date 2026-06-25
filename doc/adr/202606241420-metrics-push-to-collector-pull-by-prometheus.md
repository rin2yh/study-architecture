# ADR-202606241420: メトリクスはアプリから Alloy へ push し、Prometheus は Alloy を scrape する

- Status: Accepted
- Date: 2026-06-24
- Relates to: ADR-[[202606241356]] (可観測性スタック OTel + Grafana Alloy + Grafana), ADR-[[202606170909]] (顧客系/運用系の DB・network 分離), ADR-[[202606240522]] (Step 3 ドメインごと DB 分割)

## Context

ADR-[[202606241356]] で可観測性スタックを OpenTelemetry + Grafana Alloy + Grafana (Tempo/Loki/
Prometheus) に決め、トレースはアプリから Alloy へ OTLP、ログは stdout を Alloy が収集する方針を確定した。
残る論点が **メトリクス (Step 4-3 / #62) を Prometheus へどう取り込むか**で、push と pull の二系統がある。

- **pull**: 各サービスが `/metrics` を公開し、Prometheus が定期的に scrape する (Prometheus の伝統的方式)。
- **push**: アプリの OTel SDK が一定間隔で送り出す (OTLP)。

メトリクスの収集方式は全サービスの計装・Alloy 設定・Prometheus 設定・compose に波及し、後から変えると
横断作業になるため、4-3 着手前に単独で決め切る (ADR 規約の 2 条件を満たす)。

## Decision

**アプリ → Alloy は OTLP push、Alloy → Prometheus は scrape (pull)** の 2 段にする。Alloy が `/metrics` を
公開し、Prometheus はその 1 点を scrape する。

```
各サービス ─OTLP push→ Grafana Alloy ─[/metrics]←scrape─ Prometheus → Grafana
```

- **アプリ側は push に統一**: トレース (ADR-[[202606241356]]) と同じ OTLP で Alloy へ送る。
  `shipping-worker` のような短命・非 HTTP の consumer も送れる。各サービスは Alloy への egress だけで
  よく、顧客系/運用系の network 分離 (ADR-[[202606170909]]) と相性が良い。
- **収集の最終段は pull**: Prometheus は OTLP 直接受信ではなく**枯れた scrape**で取り込む。scrape 対象は
  Alloy 1 つに集約され、Step 3 (ADR-[[202606240522]]) の DB 分割で増えた各サービスを列挙する
  サービスディスカバリが要らない。
- **各サービスの死活は scrape で見ない**: pull が自動で付与する target-up 監視は使わず、既存の
  `GET /healthz` と compose / LB のヘルスチェックで見る。Alloy 自体の死活は scrape 成否で分かる。

## Consequences

- **push の利点と pull の利点を両取り**: アプリ側はトレースと一貫した push、収集の最終段は
  Prometheus がもっとも得意でこなれた scrape。学習用途で「枯れた経路」を選べる。
- **scrape 対象が Alloy に集約**: サービスが増えても Prometheus の scrape 設定はほぼ不変。
- **死活は別系統**: 各サービスの up/down は scrape では分からず、healthz / LB に委ねる前提になる。
  アプリが落ちることは稀で、LB のヘルスチェックで足りるという判断。
- **Alloy に集約点が寄る**: Alloy が落ちるとメトリクスの取り込みが止まる。トレース・ログと同じ
  集約点なので監視対象は増えないが、Alloy の可用性が効く。

## Alternatives considered

- **アプリから Prometheus へ直接 push (remote_write / Prometheus の OTLP receiver)**: Alloy の scrape 段が
  減り構成は単純になるが、Prometheus 3.0 で GA したばかりの新しめの経路に乗る。学習用途では枯れた
  scrape を選び、Alloy を Prometheus に scrape させる。
- **各サービスを Prometheus が直接 scrape する純 pull**: Prometheus の王道だが、全サービスのサービス
  ディスカバリ・到達経路が要り (DB 分割でサービスが多い)、`shipping-worker` のような短命・非 HTTP
  プロセスを取りこぼす。メトリクスだけトレース/ログと別 paradigm になり、ADR-[[202606241356]] の
  「Alloy 1 本道」の旨味を削ぐ。
