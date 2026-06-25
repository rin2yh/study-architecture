# ADR-202606251000: メトリクスは Alloy から Prometheus へ OTLP で push する

- Status: Accepted
- Date: 2026-06-25
- Supersedes: ADR-[[202606241420]] (Alloy→Prometheus の取り込み方式のみ)
- Relates to: ADR-[[202606241356]] (可観測性スタック)

## Context

ADR-[[202606241420]] は Alloy→Prometheus を「Alloy が `/metrics` を公開し Prometheus が scrape」と決めた。
4-3 実装時に計測すると前提が成立しない:

- Grafana Alloy v1.7.5 の `otelcol.exporter.prometheus` は OTLP→Prometheus 変換のみで、scrape 用 HTTP
  エンドポイントを公開しない (公開機能の要望 grafana/alloy#657 は not planned で close)。
- Alloy が `/metrics`(:12345) で出すのは Alloy 自身の内部メトリクスだけで、アプリの RED は含まれない。

## Decision

Alloy→Prometheus は **OTLP push**。Alloy の `otelcol.exporter.otlphttp` → Prometheus ネイティブ OTLP
receiver (`--web.enable-otlp-receiver`) へ送る。

- アプリ→Alloy が OTLP push、Alloy→Tempo も OTLP なので、端から端まで OTLP で揃い変換段が消える。
- ADR-[[202606241420]] の「アプリ→Alloy は OTLP push」「収集役は Alloy 一本」は維持。覆すのは最終段の
  取り込み方式 (scrape) だけ。

## Consequences

- OTLP 一貫で `otelcol.exporter.prometheus` の変換ロスを避けられる。
- push 受けは時刻順を保証しないため Prometheus 側で out-of-order を許容する (`out_of_order_time_window`)。
- scrape の `up`/staleness による半自動の死活監視は使えないが、死活は元々 `/healthz` で見る (ADR-[[202606241420]])。

## Alternatives considered

- Alloy が `remote_write` で push → Grafana 系の枯れた本流。OTLP 一貫を採り見送り。
- アプリを直接 scrape (promhttp) → ADR-[[202606241420]] が却下済み (`shipping-worker` 取りこぼし・OTLP push を捨てる)。
