---
name: verify-alerts
description: "docker デーモンが無いエージェント環境でも Prometheus + Grafana を実際に起動し、infra/o11y/alerting の Grafana-managed alert が本当に firing まで行くかを確認してスクリーンショットを撮る。ユーザーが「アラートが本当に鳴るか確認して」「Grafana のスクショが見たい」「provisioning が効いてるか確かめて」「ダッシュボードを実際に見せて」などと言ったとき、またアラートルールやメトリクス計装を追加・変更した直後のセルフ点検で必ず使う。docker が使えないからと諦めて YAML の目視確認で済ませない (mirror.gcr.io からイメージを取れる)。メトリクス名やラベルが OTLP 変換後にどうなるかを実測したいときも同じ手順が使える。"
argument-hint: "[確認したいアラート名・ルール uid・ダッシュボード (任意)]"
allowed-tools:
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - Bash
---

# アラート発火の実地検証

`infra/o11y/alerting/*.yaml` を書いただけでは「鳴る」保証がない。壊れるのはたいてい YAML の外側:

- メトリクス名とラベルは **OTLP 変換後** に決まる (`messaging.dlq.depth` → `messaging_dlq_depth`)。式が 1 文字違えば永久に NoData
- annotation のテンプレートは**評価時に**壊れる (`$values.B.Value` が nil で `humanizePercentage` が失敗する等)
- `for:` と閾値は、実際に系列を跨がせないと合っているか分からない

CLAUDE.md の「推測するな、計測せよ」をアラートに適用したのがこのスキル。**docker デーモンが無くても実際に立てられる**ので、設定ファイルの目視確認で終わらせない。

## この環境の制約 (先に知っておく)

| 事実 | 対処 |
| --- | --- |
| docker デーモンが無い | イメージを手で取って chroot で走らせる (`scripts/pull-oci-image.sh`) |
| `dl.grafana.com` / Docker Hub の blob CDN はネットワークポリシーで 403 | **`mirror.gcr.io`** (Docker Hub の pull-through cache) は認証なしで通る |
| GitHub release の資材 (`objects.githubusercontent.com`) は通る | Prometheus はここからバイナリを取る |
| フォアグラウンドの `sleep` は禁止 | 待ちは `run_in_background: true` のポーリングループにする |
| `pkill -f <pattern>` は**自分のシェルのコマンドラインにも当たって落ちる** | PID 指定か `pkill -x <実行ファイル名>` を使う |

バージョンは `compose.yaml` の image タグに合わせる (違う版で確認しても再現にならない)。

## 手順

作業ディレクトリはスクラッチパッド配下に取る。リポジトリを汚さない。

### 1. Prometheus を立てる

compose と同じ版のバイナリを取り、**リポジトリの設定をそのまま**使う。OTLP 受信は明示的に有効化が要る。

```bash
curl -sSL -o prom.tar.gz https://github.com/prometheus/prometheus/releases/download/v3.5.0/prometheus-3.5.0.linux-amd64.tar.gz
tar xzf prom.tar.gz
nohup ./prometheus-3.5.0.linux-amd64/prometheus \
  --config.file=/home/user/study-architecture/infra/o11y/prometheus.yaml \
  --web.enable-otlp-receiver --storage.tsdb.path=./data --web.listen-address=127.0.0.1:9090 &
```

### 2. メトリクスを流す

**本番の計装コードをそのまま呼ぶ**のが肝心。ここで手書きのダミー値を送ると、確認したかった「名前とラベルが式と一致するか」が検証できない。

一時プログラムの置き場所と依存に注意:

- `server/internal/...` を import するため、プログラムは **`server/` 配下** に置く (リポジトリ直下だと internal 制約で弾かれる)
- Prometheus の OTLP receiver は **HTTP のみ**。アプリが使う `otlpmetricgrpc` は届かないので `otlpmetrichttp` を一時的に `go get` する (エンドポイントは `127.0.0.1:9090`、パスは `/api/v1/otlp/v1/metrics`)
- DB や Redis が要る計装は `miniredis` 等のインプロセス実装で足りることが多い
- `resource` の `service.name` が Prometheus の `job` ラベルになる

例 (DLQ 滞留 gauge を確認したときの構成): miniredis に DLQ ストリームを作って数件 XADD し、`redisx.ObserveDLQDepth` を本番のまま呼んで 15 分間 push し続ける。

終わったら**必ず**一時プログラムを消し `go.mod` / `go.sum` を戻す。最後に `git status` がクリーンであることを確認する。

### 3. Grafana を立てる

```bash
.claude/skills/verify-alerts/scripts/pull-oci-image.sh grafana/grafana 12.2.0 "$WORK/grafana-root"
.claude/skills/verify-alerts/scripts/start-grafana.sh "$WORK/grafana-root" /home/user/study-architecture
```

`start-grafana.sh` は compose と同じ provisioning (datasources / alerting / dashboards) を rootfs へ写し、datasource の URL だけ `prometheus:9090` → `127.0.0.1:9090` に差し替える。匿名 Admin も compose と同じなのでログイン画面は出ない。

起動ログに `Failed to install plugin ... Forbidden` が並ぶが、プラグイン取得が塞がれているだけで検証には影響しない。

### 4. firing まで待つ

状態は API で見る。`for:` の分だけ `pending` に留まってから `firing` に変わる。**バックグラウンドで**待つ:

```bash
until curl -s "http://127.0.0.1:3000/api/prometheus/grafana/api/v1/rules" \
  | jq -e '.data.groups[].rules[] | select(.name=="<ルール名>" and .state=="firing")' >/dev/null; do sleep 15; done
```

このとき `Provisioned` 扱いになっているか (= リポジトリの YAML が読まれたか) も確認する。UI から編集できない旨のバナーが出ていれば provisioning 経由。

### 5. スクリーンショットを撮る

```bash
NODE_PATH=/opt/node22/lib/node_modules node .claude/skills/verify-alerts/scripts/screenshot.js <url> <out.png> [waitMs]
```

見せる価値がある URL:

| URL | 何が写るか |
| --- | --- |
| `/alerting/grafana/<rule-uid>/view` | 式 A → Reduce B → Threshold C の連鎖と各系列の値。**まずこれ** |
| `/alerting/grafana/<rule-uid>/view?tab=instances` | annotation が展開された実際の通知文面 |
| `/alerting/list` | firing / normal の全体像 |

`<rule-uid>` は YAML の `uid` そのもの。

### 6. 後片付け

Prometheus / Grafana / メトリクス送出プロセスを止め (PID 指定)、一時プログラムと `go.mod` の変更を戻す。作業ディレクトリはスクラッチパッドなので残してよい。

## 報告のしかた

撮れた画面が何を証明しているかを書く。「アラートが firing した」だけでなく、**実測できたメトリクス名とラベル**を添えると、式の正しさまで示せる。

Grafana をどうしても起動できなかった場合は、同じ式・同じ `for:`・同じ annotation を Prometheus 単体のルールファイルに移して `/alerts` を撮る手もある。ただしその場合は **「Grafana の画面ではない」と明記する**。等価物であって同じものではない。

## つまずいたときの勘所

| 症状 | 原因 |
| --- | --- |
| `page.goto` が networkidle で 30s タイムアウト | Grafana/Prometheus の UI はポーリングし続ける。`domcontentloaded` + 固定待ちにする (同梱スクリプトは対応済み) |
| Playwright が chromium を見つけられない | ディレクトリはバージョン付き (`/opt/pw-browsers/chromium-<n>/`)。決め打ちしない |
| ルールが `NoData` のまま | 式のメトリクス名が OTLP 変換後の実体と違う。`/api/v1/query` で実際の系列を引いて突き合わせる |
| 無関係のルールがエラーを吐く | そのルールが要求するメトリクスを流していないだけ (例: HTTP を流さずに RED アラートを見ると template error)。確認対象でなければ無視してよい |
| `chroot` した Grafana が即死 | rootfs の展開が不完全。レイヤを manifest の順に重ねられているか確認する |
