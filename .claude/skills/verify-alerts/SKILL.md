---
name: verify-alerts
description: "compose の可観測性スタック (Prometheus + Grafana + Alloy) をエージェント環境で実際に起動し、infra/o11y/alerting の Grafana-managed alert が本当に firing まで行くかを確認してスクリーンショットを撮る。ユーザーが「アラートが本当に鳴るか確認して」「Grafana のスクショが見たい」「provisioning が効いてるか確かめて」「ダッシュボードを実際に見せて」などと言ったとき、またアラートルールやメトリクス計装を追加・変更した直後のセルフ点検で必ず使う。docker デーモンは既定で動いていないが自分で起動でき、イメージも mirror.gcr.io から取れるので、「docker が無いから」と諦めて YAML の目視確認で済ませない。メトリクス名やラベルが OTLP 変換後にどうなるかを実測したいときも同じ手順が使える。"
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

CLAUDE.md の「推測するな、計測せよ」をアラートに適用したのがこのスキル。**compose のスタックはこの環境でもそのまま動く**ので、設定ファイルの目視確認で終わらせない。

## この環境の事実

| 事実 | 対処 |
| --- | --- |
| docker デーモンは起動していないが、`dockerd` は入っている | `nohup dockerd &` で自分で上げる。10 秒ほどで `docker info` が通る |
| Docker Hub 直の pull は blob の CDN で 403 になる | `mirror.gcr.io/<image>` から引いて `docker tag` で compose のイメージ名に付け替える |
| フォアグラウンドの `sleep` は禁止 | 待ちは `run_in_background: true` のポーリングループにする |
| `pkill -f <pattern>` は**自分のシェルのコマンドラインにも当たって落ちる** | PID 指定か `pkill -x <実行ファイル名>` を使う |
| 一時的に書く検証コードにもコメント規約のフックが効く | 使い捨てでもコメントは why だけ。what を書くと弾かれて往復が増える |

## 手順

作業ファイルはスクラッチパッド配下に置く。リポジトリは読むだけで、汚さない。

### 1. スタックを上げる

compose をそのまま使う。**確認したいのはリポジトリが配る設定そのもの**なので、設定を書き写した自前の起動は最後の手段にする。

```bash
nohup dockerd > "$WORK/dockerd.log" 2>&1 &   # 起動済みなら不要
for img in prom/prometheus:v3.5.0 grafana/grafana:12.2.0 grafana/alloy:v1.7.5; do
  docker pull -q "mirror.gcr.io/$img" && docker tag "mirror.gcr.io/$img" "$img"
done
docker compose -f compose.yaml -f "$WORK/verify-ports.yml" --profile observability \
  up -d --no-build --no-deps prometheus grafana alloy
```

イメージのタグは `compose.yaml` から読む (違う版で確認しても再現にならない)。`--no-deps` は tempo / loki を巻き込まないため。トレースやログまで見たいなら足す。

compose はホストへ Grafana (3000) しか公開しないので、検証プロセスから直接叩く分だけ override で開ける:

```yaml
# $WORK/verify-ports.yml
services:
  alloy:
    ports: ["4317:4317"]
  prometheus:
    ports: ["9090:9090"]
```

### 2. メトリクスを流す

**本番の計装コードをそのまま呼ぶ**のが肝心。ここで手書きのダミー値を送ると、確認したかった「名前とラベルが式と一致するか」が検証できない。

`server/` 配下に使い捨てのプログラムを置き、`otelx.Setup` + 対象の計装関数を呼ぶだけにする (`server/internal/...` を import するので、リポジトリ直下では internal 制約に弾かれる)。宛先は **Alloy** にする:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4317 OTEL_EXPORTER_OTLP_INSECURE=true ./probe &
```

Alloy 経由にすると、アプリ本来の `otlpmetricgrpc` がそのまま使えて **`go.mod` を触らずに済む**。しかも取り込み経路 (アプリ → Alloy → Prometheus) が本番と同じになる。Prometheus の OTLP receiver へ直接送る道もあるが、そちらは HTTP のみで `otlpmetrichttp` の一時追加が要るうえ、Alloy 段のマスキングを飛ばすので忠実度が落ちる。

- DB や Redis が要る計装は `miniredis` 等のインプロセス実装で足りることが多い
- `resource` の `service.name` が Prometheus の `job` ラベルになる。複数サービスを再現したいならプロセスを分ける
- 閾値を跨がせる値の作り方は対象次第 (例: p95 なら「2 割だけ 1〜1.5s、残りは 50ms」のテール偏重にすると、平均は閾値以下のまま p95 だけ超える)
- 別の検証が並行しているときは、リポジトリをスクラッチパッドへ複製してそこでビルドする (`go.mod` の取り合いを避ける)

終わったら一時プログラムを消し、`git status` がクリーンであることを確認する。

### 3. firing まで待つ

状態は API で見る。`for:` の分だけ `pending` に留まってから `firing` に変わる。**バックグラウンドで**待つ:

```bash
until curl -s "http://127.0.0.1:3000/api/prometheus/grafana/api/v1/rules" \
  | jq -e '.data.groups[].rules[] | select(.name=="<ルール名>" and .state=="firing")' >/dev/null; do sleep 15; done
```

`Provisioned` 扱いになっているか (= リポジトリの YAML が読まれたか) も確認する。UI から編集できない旨のバナーが出ていれば provisioning 経由。

### 4. スクリーンショットを撮る

```bash
NODE_PATH=/opt/node22/lib/node_modules node .claude/skills/verify-alerts/scripts/screenshot.js <url> <out.png> [waitMs]
```

見せる価値がある URL:

| URL | 何が写るか |
| --- | --- |
| `/alerting/grafana/<rule-uid>/view` | 式 A → Reduce B → Threshold C の連鎖と各系列の値。**まずこれ** |
| `/alerting/grafana/<rule-uid>/view?tab=instances` | annotation が展開された実際の通知文面 |
| `/alerting/grafana/<rule-uid>/view?tab=history` | pending → alerting の遷移。`for:` が効いている証拠になる |
| `/alerting/list` | firing / normal の全体像 |

`<rule-uid>` は YAML の `uid` そのもの。

### 5. 後片付け

`docker compose ... down` でコンテナを落とし、検証プロセスを止め (PID 指定)、一時プログラムを消す。`git status` がクリーンであることを最後に確認する。

## 報告のしかた

撮れた画面が何を証明しているかを書く。「アラートが firing した」だけでなく、**実測できたメトリクス名とラベル**と、**pending → firing の実時刻**を添えると、式と `for:` の正しさまで示せる。

スタックをどうしても起動できなかった場合は、同じ式・同じ `for:`・同じ annotation を Prometheus 単体のルールファイルに移して `/alerts` を撮る手もある。ただしその場合は **「Grafana の画面ではない」と明記する**。等価物であって同じものではない。

## つまずいたときの勘所

| 症状 | 原因 |
| --- | --- |
| 起動直後に `Normal (NoData)` のゴーストインスタンスが出る | まだ系列が無い 1 評価分だけ。次の評価で消える (`noDataState: OK` なら誤検知にならない) |
| Grafana の画面が真っ黒 / "failed to load its application files" | ホストのロケール由来で bootstrap の `Intl.NumberFormat` が落ちている。`LANG=en_US.UTF-8` を与える |
| ルールが `NoData` のまま | 式のメトリクス名が OTLP 変換後の実体と違う。`/api/v1/query` で実際の系列を引いて突き合わせる |
| データを止めたのに firing が続く | `relativeTimeRange` の窓 (既定 10 分) に最後の点が残る限り `reducer: last` が拾い続ける。仕様であって不具合ではない |
| alert instance の Value が `1e+00` | それは Threshold C の真偽値。件数や実測値は `$values.B.Value` (Reduce の出力) 側 |
| 無関係のルールがエラーを吐く | そのルールが要求するメトリクスを流していないだけ (例: HTTP を流さずに RED を見ると template error)。確認対象でなければ無視してよい |
| `playwright install` が 403 | `cdn.playwright.dev` は塞がれている。`/opt/pw-browsers` のプリインストール版を使う (同梱スクリプトは対応済み) |
| `page.goto` が networkidle で 30s タイムアウト | Grafana/Prometheus の UI はポーリングし続ける。`domcontentloaded` + 固定待ちにする (同梱スクリプトは対応済み) |

## docker が使えないときのフォールバック

`dockerd` が上がらない環境向けに、イメージを手で展開して chroot で走らせる道も用意してある。compose を経由しないぶん忠実度は落ちる (Alloy 段を飛ばす、datasource の URL を書き換える) ので、**docker が動くならこちらは使わない**。

```bash
scripts/pull-oci-image.sh grafana/grafana 12.2.0 "$WORK/grafana-root"
scripts/start-grafana.sh "$WORK/grafana-root" /home/user/study-architecture   # GRAFANA_PORT で移せる
```

この経路では Prometheus をバイナリで別途起動し (`--web.enable-otlp-receiver`)、メトリクスは `otlpmetrichttp` で `/api/v1/otlp/v1/metrics` へ直接送ることになる (`go.mod` の一時変更が要る。終わったら戻す)。
