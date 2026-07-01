# ADR-202606301100: checkout の配送先は BFF が解決し order へ値で渡す (order は member を引かない)

- Status: Accepted
- Date: 2026-06-30
- Relates to: ADR-[[202606261704]] (配送先スナップショット), ADR-[[202606190900]] (横断データの権威は持ち主), ADR-[[202606230930]] (BFF が認証コンテキストを集約), GitHub #89

## Context

- 当初は order が checkout 時に member の `GetAddress` を引いて住所を確定していた (product を引くのと同型)。だが order が member に依存する辺が増える。
- store BFF はすでに認証コンテキストの集約点で (ADR-[[202606230930]])、配送先選択 UI のために member の住所帳を引いている。住所の解決を二重に行っていた。

## Decision

配送先の解決は BFF に集約し、order へは値で渡す。

- **store BFF (checkout action) が `shippingAddressId` を member の `GetAddress` で解決し、住所の値を checkout リクエストに同梱する**。order は受け取った値を注文時点でスナップショットするだけで member を呼ばない。
- 権威は依然 member。BFF はサーバ側で member から引くため、ブラウザが値を詐称する経路にはならない (ADR-[[202606190900]] の「権威は持ち主」を BFF 経由で満たす)。
- 決め手: order の依存辺を減らし、member 解決を集約点 (BFF) に一本化する。

## Consequences

- order は product / payment / inventory のみに依存し、member クライアント / gateway / `MEMBER_API_URL` を持たない。
- 住所詐称の懸念は BFF がサーバ側で member から引くことで担保 (価格と違い住所は値の取り違えで利益が出る性質でもない)。
- checkout の公開契約が `shippingAddressId` から `shippingAddress` (値) に変わる。

## Alternatives considered

- **order が member.GetAddress を引く** (初稿。product スナップショットと同型): 権威は最も素直だが、BFF が既に持つ member 解決を order に重複させ、order→member の依存辺が増える。BFF が信頼できるサーバ側集約点 (ADR-[[202606230930]]) であるため、解決を BFF に寄せて差し替えた。
- **ブラウザが住所値を直接 order へ送る**: BFF/order とも member を引かずに済むが、権威がクライアントに移る。BFF 経由なら server-side で member から引けるため不採用。
