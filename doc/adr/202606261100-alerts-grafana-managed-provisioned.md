# ADR-202606261100: アラートは Grafana-managed alerting で provisioning する

- Status: Accepted
- Date: 2026-06-26
- Relates to: ADR-[[202606241356]] (可観測性スタック), ADR-[[202606251000]] (メトリクス取り込み), ADR-[[202606250141]] (マスキング)

## Context

4-1〜4-6 で RED メトリクスと固定ダッシュボードは揃ったが、閾値超過を「人が画面を見ていないと気づけない」。
エラー率・レイテンシが異常域に入ったら検知する仕組みが要る (#73)。これは計装 (可観測性) を超えて運用監視に
踏み込む部分で、実現方式は複数コンポーネントをまたぎ代替案にトレードオフがある = ADR 基準 (`.claude/rules/adr.md`)。

## Decision

**Grafana-managed alerting** を採り、ルールは `infra/o11y/alerting/` に YAML で置いて provisioning する。
評価データソースは既設の Prometheus (RED)。

- Grafana は既に起動・datasource 設定済みで **追加コンテナが不要**。ADR-[[202606241356]] の「収集役 Alloy 一本・
  部品を増やさない」と整合する。
- ルール・contact point・notification policy をすべてコードに置き、手動作成に依存しない (ダッシュボードと同じ方針)。
- 通知先はローカル完結・費用ゼロ前提で実宛先 (Slack/メール) を持たない。アラート状態を **Grafana UI で見える**
  ところまでを基本線とし、contact point は無音の既定 (email) のままにする。発火は Alerting 画面で確認する。
- 評価は observability profile に隔離し、Alloy/backend 不在でもアプリへ影響しない (既存方針)。

## Consequences

- RED の Error / Duration が閾値を割ると Grafana 上で firing になり、画面で気づける。
- アラート定義がリポジトリから再現でき、Grafana を作り直しても同じルールが復元される。
- 実通知 (Slack 等) は無い。学習用途では UI 可視化で足り、宛先が要るときに contact point を足せばよい。

## Alternatives considered

- **Prometheus `rule_files` + Alertmanager** → Prometheus ネイティブだが Alertmanager コンテナが増える。
  現 `infra/o11y/prometheus.yaml` は OTLP 取り込みのみで `rule_files` も未設定。部品を増やさない方針に反するため見送り。
- **Alloy の `prometheus.rules` 相当で評価** → Alloy にアラート評価を持たせると収集と監視が混ざる。Grafana に寄せる。
