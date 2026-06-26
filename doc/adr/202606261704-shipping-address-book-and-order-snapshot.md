# ADR-202606261704: 配送先住所は member の住所帳で持ち、注文時に order/shipment へスナップショットする

- Status: Accepted
- Date: 2026-06-26
- Relates to: ADR-[[202606190900]] (横断データの注文時スナップショット = 同型), ADR-[[202606211200]] (settled イベントで配送手配), ADR-[[202606231000]] (schema 所有権), GitHub #89

## Context

- 配送を手配する (ADR-[[202606211200]]) のに宛先がどこにも無い。member は `email / displayName` のみ、order は届け先を受け取らず、`shipping.shipments` は宛先列を持たない。宛先不明のまま shipment を作っている (#89)。

## Decision

住所を member の住所帳で持ち、注文時にスナップショットする。

- **住所は member が住所帳 (複数住所) として所有する** (会員に属する情報。ADR-[[202606231000]])。
- **checkout で配送先 (住所帳の1件) を選び、注文時点の住所を order/shipment へ値でスナップショットする** (商品名・単価のスナップショット ADR-[[202606190900]] と同型)。後からの住所変更を既存注文に遡及させない。
- **shipment は settled イベント経由で宛先スナップショットを受け取り保持する**。

## Consequences

- 注文後に member が住所を変えても、既存注文・shipment の宛先は不変。
- 住所帳と注文スナップショットで住所が二重化するが、これは意図した非正規化 (ADR-[[202606190900]] と同じトレードオフ)。
- settled イベント payload に宛先が増える (`server/internal/paymentevent/`)。shipping が order を引く経路にするかは実装で詰める。

## Alternatives considered

- **member に単一住所**: 最小だが複数届け先を扱えない。住所帳の方が自然で実装差も小さい。
- **shipment から member を住所参照 (正規化)**: 住所変更が過去注文に遡及し、配送実態と食い違う (ADR-[[202606190900]] のスナップショット原則に反する)。
- **住所を order が所有**: 住所は会員のプロフィールであり order の関心ではない。
