---
name: grafana-screenshot
description: "compose の可観測性スタック (Grafana + Prometheus + Alloy) をこの環境で実際に起動し、ダッシュボードやアラート画面のスクリーンショットを撮って見せる。ユーザーが「Grafana のスクショ見たい」「ダッシュボードを見せて」「アラートが鳴ってるところを見たい」「画面で確認して」「provisioning が効いてるか確かめて」などと言ったときに必ず使う。docker デーモンは既定で止まっているが自分で起動でき、イメージも mirror.gcr.io から取れるので、「docker が無いから無理」と諦めて設定ファイルの説明で代替しない。"
argument-hint: "[撮りたいダッシュボード名・アラート名 (任意)]"
allowed-tools:
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - Bash
---

# Grafana の画面を撮る

グラフやアラートは、説明を並べるより 1 枚見せた方が早い。この環境は docker デーモンが止まっているだけで、**compose のスタックはそのまま動く**。設定ファイルの引用で代替しない。

## 手順

作業ファイルはスクラッチパッド配下に置く。リポジトリは読むだけで、汚さない。

### 1. スタックを上げる

```bash
.claude/skills/grafana-screenshot/scripts/up.sh /home/user/study-architecture "$WORK"
```

`dockerd` の起動 (止まっていれば) → `mirror.gcr.io` からイメージを取って compose のイメージ名へ付け替え → `docker compose --profile observability up` までやる。イメージが温まっていれば 1 分弱で `http://127.0.0.1:3000` が匿名 Admin で開く (Prometheus は `:9090`、Alloy の OTLP は `:4317`)。

使うのは**リポジトリの compose 定義そのもの**なので、写っている画面は provisioning された datasource / dashboards / alerting をそのまま反映している。tempo と loki は `--no-deps` で外してあるので、トレースやログの画面が要るときは足す。

### 2. 見せたいデータを流す (要るときだけ)

パネルのレイアウトを見せるだけなら不要。値やアラートの発火を見せたいなら、**本番の計装コードをそのまま呼ぶ**使い捨てプログラムを `server/` 配下に置いて Alloy へ送る (`server/internal/...` を import するため、リポジトリ直下では internal 制約に弾かれる)。

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4317 OTEL_EXPORTER_OTLP_INSECURE=true ./probe &
```

- `otelx.Setup` + 対象の計装関数を呼ぶだけでよい。Alloy 経由なのでアプリ本来の `otlpmetricgrpc` が使え、`go.mod` を触らずに済む
- 手書きのダミー系列を送ると「式とラベルが噛み合っているか」が確かめられなくなる。名前や属性を自分で書かない
- DB や Redis が要る計装は `miniredis` 等のインプロセス実装で足りることが多い
- `resource` の `service.name` が `job` ラベルになる。複数サービスを写したいならプロセスを分ける
- 使い捨てコードにもコメント規約のフックが効く。コメントは why だけにする

流し終えたらプログラムを消し、`git status` がクリーンなことを確認する。

### 3. 撮る

```bash
NODE_PATH=/opt/node22/lib/node_modules node .claude/skills/grafana-screenshot/scripts/screenshot.js <url> <out.png> [waitMs]
```

| 撮りたいもの | URL |
| --- | --- |
| ダッシュボード | `/d/<uid>?kiosk&from=now-15m&to=now` (`kiosk` でナビが消え、パネルだけになる) |
| ダッシュボード一覧 | `/dashboards` |
| アラートルールの中身 | `/alerting/grafana/<rule-uid>/view` — 式 A → Reduce B → Threshold C の連鎖と各系列の値 |
| 発火時の通知文面 | `/alerting/grafana/<rule-uid>/view?tab=instances` — annotation が展開された姿 |
| 状態遷移 | `/alerting/grafana/<rule-uid>/view?tab=history` — pending → alerting の時刻が読める |
| アラート全体 | `/alerting/list` |

`<uid>` はダッシュボード JSON / アラート YAML の `uid`。分からなければ `curl -s 'http://127.0.0.1:3000/api/search?type=dash-db' | jq` で引く。

**アラートの発火を撮るとき**は `for:` の分だけ待つ必要がある。フォアグラウンドの `sleep` は使えないので、**バックグラウンド**のポーリングで待つ:

```bash
until curl -s "http://127.0.0.1:3000/api/prometheus/grafana/api/v1/rules" \
  | jq -e '.data.groups[].rules[] | select(.name=="<ルール名>" and .state=="firing")' >/dev/null; do sleep 15; done
```

### 4. 片付ける

`docker compose -f compose.yaml -f "$WORK/ports.yml" --profile observability down` でコンテナを落とし、データ送出プロセスを止める (PID 指定。`pkill -f` は自分のシェルにも当たって落ちる)。

## 見せ方

画面を貼るだけで終わらせず、**その画面が何を示しているか**を 1〜2 行添える。アラートなら「pending → firing の実時刻」と「実測できたメトリクス名とラベル」を書くと、`for:` と式の正しさまで示せる。

データを流していないダッシュボードは全パネル `No data` になる。それでも意味があるとき (レイアウトの確認) はそう言い、値を見せるべき場面なら手順 2 に戻る。

## つまずいたときの勘所

| 症状 | 原因 |
| --- | --- |
| Grafana の画面が真っ黒 / "failed to load its application files" | ホストのロケール由来で bootstrap の `Intl.NumberFormat` が落ちている。`LANG=en_US.UTF-8` を与える |
| `page.goto` が networkidle で 30s タイムアウト | Grafana の UI はポーリングし続ける。`domcontentloaded` + 固定待ちにする (同梱スクリプトは対応済み) |
| `playwright install` が 403 | `cdn.playwright.dev` は塞がれている。`/opt/pw-browsers` のプリインストール版を使う (同梱スクリプトは対応済み) |
| パネルが全部 `No data` | データを流していないだけ。手順 2 |
| 起動直後のアラートに `Normal (NoData)` のゴーストインスタンス | まだ系列が無い 1 評価分だけ。次の評価で消える |
| データを止めたのに firing が続く | `relativeTimeRange` の窓 (既定 10 分) に最後の点が残る限り `reducer: last` が拾い続ける。仕様であって不具合ではない |
| alert instance の Value が `1e+00` | それは Threshold の真偽値。実測値は `$values.B.Value` (Reduce の出力) 側 |
| ポートが埋まっている | 別の検証が動いている。`docker compose ps` で確認し、落とすか終わるのを待つ |
