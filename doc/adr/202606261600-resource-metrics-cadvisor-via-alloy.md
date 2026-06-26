# ADR-202606261600: リソースメトリクスは cAdvisor を Alloy で収集し既存 OTLP 経路へ相乗りさせる

- Status: Accepted
- Date: 2026-06-26
- Relates to: ADR-[[202606241356]] (収集役は Alloy 一本), ADR-[[202606251000]] (Alloy→Prometheus は OTLP push), ADR-[[202606250141]] (マスキング)

## Context

4-1〜4-6 でアプリのテレメトリ (トレース / ログ / RED / DB プール統計) と Grafana ダッシュボードは揃ったが、
**リソース (インフラ) 系メトリクスは誰も収集していない**。Go アプリは otelgin / otelhttp / otelpgx しか
計装しておらず、CPU / メモリ / ディスクを出していない。「どのサービスが食っているか」を画面で追えない。

収集方式は複数コンポーネント (exporter / 収集役 / Prometheus) をまたぎ、案 A/B にトレードオフがあるため
ADR にする (`.claude/rules/adr.md`)。

## Decision

**cAdvisor を Alloy の `prometheus.exporter.cadvisor` で収集し、`otelcol.receiver.prometheus` ブリッジで
既存の OTLP push 経路 (masking → otlphttp.prometheus, ADR-[[202606251000]]) へ相乗り**させる。

- コンテナ単位で CPU / メモリ / ディスク / ネットワークが取れ、**サービス別**に追える。
- ブリッジで OTLP に寄せるので Prometheus への入口は 1 つ(OTLP receiver)のまま。収集役 Alloy 一本
  (ADR-[[202606241356]]) と整合し、masking 段 (ADR-[[202606250141]]) も共通で通る。
- コンテナ名 `name` を compose のサービス名へ relabel し、ログ側の `service` ラベルと軸を合わせる。
  生のコンテナ ID / name / image は高カーディナリティかつ識別子なので落とす (ADR-[[202606250141]])。

## Consequences

- Alloy に privileged + ホスト経路マウント (`/`, `/sys`, `/var/lib/docker`, `/var/run`) が要る。
  テレメトリ収集役のみ・本番運用ではない学習スタックなので許容する (compose にコメントで明示)。
- cAdvisor はメトリクスが多い。サービス別 4 系統 (CPU/mem/disk/net) に絞って出し、不要ラベルは relabel で畳む。
- **CI / 入れ子 docker サンドボックスでは cAdvisor がコンテナ名を解決できず** (storage driver 由来で `name`
  ラベルが付かない)、サービス別にならない。検証スクショは OrbStack / 通常 docker 前提にする (runbook)。

## Alternatives considered

- **案B: node_exporter (`prometheus.exporter.unix`) でホスト全体** → /proc・/sys を読むだけで privileged 不要
  だが **サービス別にならない** (ホスト一括)。「どのサービスが食っているか」を追う目的に合わず見送り。
- **cAdvisor を独立コンテナ + Prometheus 直 scrape** → Prometheus への入口が OTLP と scrape の 2 系統に増え、
  収集役 Alloy 一本 (ADR-[[202606241356]]) を崩す。ブリッジで OTLP に寄せる方が既存経路と噛み合う。
