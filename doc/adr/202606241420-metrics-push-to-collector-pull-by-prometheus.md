# ADR-202606241420: メトリクスはアプリから Alloy へ push し、Prometheus は Alloy を scrape する

- Status: Accepted (Alloy→Prometheus の取り込み方式は ADR-[[202606251000]] で Superseded)
- Date: 2026-06-24
- Relates to: ADR-[[202606241356]] (可観測性スタック), ADR-[[202606170909]] (network 分離)

## Context

メトリクス (4-3) を Prometheus へ取り込む方式に push (OTLP) と pull (scrape) がある。全サービスの計装・
Alloy・Prometheus 設定に波及するため、4-3 着手前に決める。

## Decision

**アプリ → Alloy は OTLP push、Alloy → Prometheus は scrape (pull)** の 2 段。Alloy が `/metrics` を公開し
Prometheus がそれを scrape する。

- アプリ側はトレースと同じ push に統一 (`shipping-worker` も送れる、egress だけで network 分離と相性良)。
- 最終段は Prometheus の枯れた scrape。対象は Alloy 1 つでサービスディスカバリ不要。
- 各サービスの死活は scrape でなく `GET /healthz` / LB で見る。

## Consequences

- push の統一性と pull の枯れた経路を両取り。
- scrape 対象が Alloy に集約。サービスが増えても設定はほぼ不変。
- Alloy が落ちると取り込みが止まる (トレース/ログと同じ集約点)。

## Alternatives considered

- アプリ → Prometheus 直接 push (remote_write / OTLP receiver) → Prometheus 3.0 で GA したばかりの
  新しめ経路。枯れた scrape を選ぶ。
- 各サービスを直接 scrape する純 pull → ディスカバリ要・`shipping-worker` を取りこぼす・別 paradigm。
