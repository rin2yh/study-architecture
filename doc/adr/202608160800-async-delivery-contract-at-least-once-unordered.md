# ADR-202608160800: 非同期イベントの受け取り契約を at-least-once + 順不同に固定する

- Status: Accepted
- Date: 2026-08-16
- Relates to: ADR-[[202608150830]] (マネージドキュー), ADR-[[202606300600]] (outbox), ADR-[[202606261214]] (冪等), ADR-[[202608160810]] (shipping の順不同耐性), GitHub #107

## Context

`deploy.replicas: 2` で計測した (compose.replicas.yaml / `mise run test:e2e:replicas`)。

- **重複**: outbox リレーは `published_at IS NULL` の取得と送出済みマークが別トランザクションなので、
  多重起動すると同じ行を複数インスタンスが送る。Postgres に 2 リレーを直接当てた計測で 300 行 →
  586 publish。定常時 (未送信が溜まっていない) はほぼ 1x だが、backlog を抱えたまま同時起動すると
  200 行 → 310 publish (1.55x) まで増えた。**欠落は 0**。
- **順序**: 標準キューは順不同で、`payment.settled` と `order.cancelled` はトピックもキューも別。
  レプリカ数に関係なく到着順は保証されない。
- 最終状態は壊れなかった (checkout → 決済確定で shipment 1 件・予約確定 1 回)。順序の穴だけは
  実測で再現したため ADR-[[202608160810]] で塞いだ。

## Decision

配送の保証を **at-least-once + 順不同**とし、consumer 側の冪等 (ADR-[[202606261214]]) で受ける。
順序保証 (FIFO / ordering key) は導入しない。

- 送出側 (outbox リレー) の多重起動を排除しない。重複は仕様として許容し、受信側で吸収する。
- consumer は「同じイベントが何度でも、どの順でも来る」前提で書く。順序に依存する処理は、順序を
  ブローカに要求せず**状態を DB に残して順不同でも収束させる**。
- レプリカを増やす変更は `mise run test:e2e:replicas` を通す。

## Consequences

- 重複ぶんの送出コストを払う。単一インスタンスに絞る仕掛け (advisory lock) も
  `FOR UPDATE SKIP LOCKED` も入れていないので、コストが問題になった時点で ADR-[[202606300600]] の
  将来案を実装する。
- 順序を要求しない分、consumer は「後から来る古いイベント」を弾く責務を持つ。新しい consumer を
  足すたびに順不同の検討が要る。
- ローカルの kumo は `sns.Subscribe` が冪等でなくレプリカごとに購読が増えるため、同じイベントが
  レプリカ数だけ余分に配送される。実 SNS は同一 topic + endpoint なら冪等なので**ローカル固有の
  乖離**であり、ローカルでの重複数をそのまま本番の見積もりに使えない。

## Alternatives considered

- **FIFO キュー / ordering key に注文 ID を使う** (ADR-[[202608150830]] が余地として残した案): 順序が
  必要なのは `payment.settled` と `order.cancelled` の 2 種だけで、しかも別トピックのため 1 つの
  ordering key にまとめられない。トピックを統合すれば購読者ごとの隔離 (ADR-[[202608150830]]) が崩れる。
- **exactly-once を狙う**: ブローカ側にその保証は無く、結局 consumer 冪等が要る。二重の仕組みになる。
- **リレーを単一インスタンスに固定する運用ルール**: レプリカ数の制約がコードに現れず、増やした瞬間に
  静かに壊れる。重複を許容して受信側で受ける方が構成変更に強い。
