# Grafana ダッシュボード

`mise up:obs` で起動する Grafana に provisioning 済みのダッシュボードの見方。役割分担は
ADR = 設計判断 / README = 概要 / 本ファイル = ダッシュボードの見方。

- 開き方: `mise up:obs` 後 http://localhost:3000 （anonymous / Admin）。Dashboards に固定で出る。
- 単一情報源: `infra/o11y/dashboards/*.json`（provider は `infra/o11y/grafana-dashboards.yaml`）。
  `allowUiUpdates: false` なので UI で編集しても保存されない。変更は JSON を直す。
- メトリクス名は Prometheus の OTLP 変換後（`UnderscoreEscapingWithSuffixes`、`infra/o11y/prometheus.yaml`）
  の実体に合わせている。

## RED — サービス別 Rate / Error / Duration (`uid: red-services`)

サービス境界をまたぐ 1 リクエストの Rate / Error / Duration を追う。`$service` 変数でサービスを絞れる。

| パネル | 見るもの |
| --- | --- |
| Rate | サービス別の req/s |
| Errors | サービス別の **5xx** 率。4xx は準正常系として除外（ADR-[[202606211520]]） |
| Duration（合算） | 選択サービス合算の p50 / p95 / p99 |
| Duration p95 | サービス別 p95 |
| Rate（ルート別） | `http_route` 別 req/s。checkout 等の遅いホップ特定に使う |
| Rate（ステータス別） | ステータスコード別 req/s |

- 元メトリクス: `http_server_request_duration_seconds_*`（otelgin / otelhttp, #62）。サービス識別は
  `job`、ステータスは `http_response_status_code`、経路は `http_route`。

## DB プール — pgxpool 統計 (`uid: db-pool`)

接続プールの飽和・待ちを追う。`$service` 変数で絞れる。

| パネル | 見るもの |
| --- | --- |
| コネクション数 | total / acquired / idle / max |
| 取得レート | acquires/s |
| 空プール待ち | プールが空で待った取得の発生率（飽和の兆候） |
| 空プール平均待ち時間 | 待った取得 1 件あたりの待ち時間 |
| 平均取得所要時間 | 取得 1 件あたりの所要時間 |
| 新規接続 / キャンセル取得 | new / canceled のレート |

- 元メトリクス: `pgxpool_*`（otelpgx `RecordStats`, Step 4-5 #71）。

## レート系パネルが空になるときの注意

メトリクスの取り込みは scrape でなくアプリからの OTLP push（既定 60s 間隔, ADR-[[202606251000]]）。
Grafana に scrape 間隔を伝えないと `$__rate_interval` が短すぎて `rate()` が 2 サンプルを跨げず
「No data」になる。これを避けるため Prometheus datasource に `timeInterval: 60s` を設定している
（`infra/o11y/grafana-datasources.yaml`）。export 間隔を変えたらこの値も合わせる。
