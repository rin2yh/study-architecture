# ADR-202607301418: 配送回数上限を超えたメッセージは group 別 DLQ へ隔離する

- Status: Accepted
- Date: 2026-07-30
- Relates to: ADR-[[202606261100]] (アラートの provisioning), ADR-[[202606261214]] (冪等性), ADR-[[202606261216]] (縮退方針), ADR-[[202606250141]] (マスキング)

## Context

- 未 ACK メッセージの引き取り (#105) で失敗は再処理されるようになったが、恒久的に処理できないメッセージは
  min-idle 間隔で無限に再試行される。一方、壊れた payload は握って ack していたため、補償・手配の欠落が
  無言で消えていた (#106)。
- 5 つの consumer ループが 2 つの stream (`payment.events` / `order.events`) を group 別に読む。

## Decision

配送回数が上限 (10) に達したメッセージは process へ通さず、`dlq:<stream>:<group>` へ複製して ack する。

- 上限判定に PEL の delivery count が要るので、引き取りは XAUTOCLAIM (配送回数を返さない) をやめ、
  XPENDING で回数を読んでから XCLAIM で値ごと引き取る。
- 退避先を stream だけでなく **group でも分ける**: 同じ stream を複数 group が読むため、混ぜると
  「どの consumer が処理できなかったのか」を復旧時に判別できない。
- 元の値はそのまま複製し、追跡用のメタだけ足す。payload は producer が載せた値のみで秘匿情報は
  混ざらない (ADR-[[202606250141]])。
- パース不能な payload も握り潰さず error を返し、この経路で DLQ へ落とす。無駄な再試行が数回走るが、
  「再配送で直る / 直らない」を handler が自己申告する仕組みを増やさずに済む。
- 監視は退避件数の counter でなく **滞留量 (XLEN) の gauge**。DLQ は誰も消費しないので、rate で見ると
  「退避が止まった = 解決」と誤って解消する。gauge 登録の失敗は consumer を止めない (ADR-[[202606261216]])。

## Consequences

- poison message は最大 10 回で本流から外れ、PEL が無限に膨らまなくなる。DLQ が空でも gauge は 0 を報告
  するので、アラートは NoData と滞留 0 を取り違えない。
- 退避は at-least-once。DLQ への複製後・ACK 前に落ちると DLQ 側に重複しうる。
- 上限到達まで min-idle × 上限 (≒ 5 分) かかる。壊れた payload の即時退避はしない。逆にこれより長い
  一過性障害では、処理できたはずのメッセージも退避される。
- DLQ は自動では減らない。中身の確認・再投入は当面 redis-cli で行う (backoffice からの操作は後続)。
- 再投入は元 stream へ戻すため、処理できていた他 group にも再配信される。二重処理は冪等性
  (ADR-[[202606261214]]) に委ねる。

## Alternatives considered

- **DLQ を stream 単位にする** (`dlq:<stream>`): ストリーム数は減るが、group を値で持たせても滞留量の系列が
  group をまたいで合算され、どの consumer の問題か切り分けられない。
- **退避件数の counter でアラート**: 実装は最小だが、滞留したまま解消するので「放置」を検知できない。
- **Redis 外 (DB テーブル等) へ退避**: 参照 UI は作りやすいが、メッセージング障害時に別依存へ書く形になり
  退避自体が失敗しうる。同一 Redis 内の別 stream なら XADD 1 回で完結する。
- **壊れた payload を即時退避**: 無駄な再試行は消えるが、「再配送では直らない」を表す sentinel error を
  handler と 2 つの読み取り経路 (新規読み / 引き取り) に通す必要があり、5 consumer に分岐が増える。
