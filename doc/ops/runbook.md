# 運用ランブック

アラート発火時など、運用で参照する手順を集約する。役割分担: 設計判断は [ADR](../adr/README.md)、
プロジェクト概要は [README](../../README.md)、ダッシュボードの見方は [dashboards](dashboards.md)。
各 Step で運用に効くものが出たらここへ追記する。

## アラート

可観測性スタック (ADR-[[202606241356]]) の Grafana-managed alert (`infra/o11y/alerting/`,
ADR-[[202606261100]])。`mise up:obs` 後、Grafana の `EC Service Alerts` フォルダ (`RED` / `Messaging`) で
状態を見る。
発火時はまず該当サービス (`job` ラベル) を特定し、トレース ⇄ ログ ⇄ メトリクスを相互に辿る。

### HTTP 5xx error rate high

- 意味: サービス (`job`) の 5xx 率が 5% を超えて 1 分継続。
- 調べ方: 該当 `job` のエラー span を Tempo で開き、`trace_id` でログ (Loki) を辿る (#61)。
  DB 起因が疑わしければ pgxpool メトリクス (#71) と DB ログを確認する。
- よくある原因: 下流 (DB / payment 同期呼び出し) の障害、デプロイ直後の不整合。
- 切り分け: `db-product` を停止すると product の 5xx を再現でき、本アラートの発火を確認できる。
- 対処: 原因のホップ (下流サービス / DB) を復旧する。

### HTTP p95 latency high

- 意味: サービス (`job`) の p95 レイテンシが 0.5s を超えて 2 分継続。
- 調べ方: 遅いトレースを Tempo で開き、どのホップ (order→payment、DB acquire 待ち 等) が
  支配的かを見る。DB プールは pgxpool の acquire 待ち / idle / total を確認する (#71)。
- よくある原因: DB プール枯渇、下流レイテンシ、想定外のクエリ件数。
- 対処: ボトルネックのホップに応じてプール上限・クエリ・下流を調整する。

### DLQ backlog present

- 意味: 配送回数の上限に達したメッセージが DLQ (`dlq:<stream>:<group>`) に 5 分以上残っている
  (ADR-[[202607301418]])。滞留量は DLQ を空にするまで下がらない。
- 調べ方: `messaging_destination_name` ラベルで退避先が分かる。中身は broker で直接見る。

  ```bash
  docker compose exec broker redis-cli XLEN dlq:payment.events:shipping
  docker compose exec broker redis-cli XRANGE dlq:payment.events:shipping - + COUNT 10
  ```

  `dlqSourceId` (元メッセージの ID) でログ・トレースを辿り、10 回とも失敗した理由を確認する。
- よくある原因: 壊れた payload (`orderId` がパース不能など)、参照先が存在しない注文、5 分以上続いた
  下流障害 (この場合は正常なメッセージも巻き込まれる)。
- 対処: 原因を直したうえで、再投入するメッセージを元 stream へ `XADD` し直し、DLQ 側は `XDEL` で消す。
  再投入は同じ stream の他 group にも配信されるが、二重処理は冪等性 (ADR-[[202606261214]]) で吸収される。
  再投入しないと決めたものも、滞留アラートを解消するため `XDEL` する。
