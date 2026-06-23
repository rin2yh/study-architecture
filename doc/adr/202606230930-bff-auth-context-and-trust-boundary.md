# ADR-202606230930: store BFF に認証コンテキストの集約と X-Member-Id の付与点を置く

- Status: Accepted
- Date: 2026-06-23
- Relates to: ADR-[[202606170900]] / ADR-[[202606170905]] / ADR-[[202606211100]] / ADR-[[202606220300]]

## Context

ADR-[[202606170900]] のロードマップ Step 1「API ファサードを足し UI 入口を集約（薄く保ち、段取りは
各サービスに残す）」に着手する。調査の結果、ファサードの機能は既に store の React Router v7 サーバ層
（loader / action）に実装されていた。`currentMemberId` による Cookie セッション検証
（ADR-[[202606211100]]）があり、サービス呼び出しもサーバ側に閉じている（ADR-[[202606170905]]）。
ADR-[[202606170905]] は「この UI サーバ層が Step 1 の BFF へ自然に育つ」と述べ、ADR-[[202606211100]] は
「`X-Member-Id` を誰が付けてよいかの信頼境界の確立は Step 1 の BFF 層へ寄せる」と残していた。

そのため新サーバを立てる段ではなく、暗黙に BFF として機能している store サーバ層を formalize する。
現状には 2 つの綻びがある:

- 認証チェックが route ごとに散在し制御フローも不統一（`orders` は未認証で `throw redirect("/login")`、
  `login` は既認証で逆向き redirect、`checkout` は error envelope を返す）。
- `X-Member-Id` の付与が `orders` のローダで手書きの header リテラルになっており、付与点が 1 か所に
  定まっていない。付け忘れや、検証していない値の混入を防ぐ構造がない。

## Decision

スコープは store のみ。新サーバ・新 `shared/api` は作らない（`shared/api` 新設は ADR-[[202606220300]]
が否定。生成クライアントの `api` パッケージはその下層として使う）。

1. **認証コンテキストの集約**: loader 用ガードを `entities/session` に置く。`requireMemberId(request)`
   は未認証なら `throw redirect("/login")`、認証済みなら `memberId` を返す。`login` の逆向きガードは
   `redirectIfAuthenticated(request)`。`checkout` の action は redirect ではなく error envelope を返す
   別仕様で、`throw redirect` を action 内で catch すると Response を壊すため、ここはガードに寄せず
   `currentMemberId` の直呼びを残す。throw / return の 2 モードを引数で切り替えるとガードが太るので、
   1 関数 1 責務に保つ。

2. **X-Member-Id 付与点の一元化**: `entities/order` に `listMyOrders(memberId)` を置き、`X-Member-Id` の
   付与をここ 1 経路に閉じる。引数は検証済み `memberId: number` のみで、`RequestInit` / header を一切
   受け取らない。これにより呼び出し側はヘッダを組む手段を持たず、ブラウザ由来の値が `X-Member-Id` に
   乗る経路がコード上から消える。order 以外も将来 `X-Member-Id` を要求しはじめたら、付与の共通部を
   `entities/session` か小さな共有 lib へ昇格する。

段取り（checkout の手順など）は order サービスに残し、BFF 層は集約と付与点に限って薄く保つ
（ADR-[[202606170900]] Step 1）。

## Consequences

- `X-Member-Id` の付与点が store BFF の 1 関数に集約され、検証済み `memberId` 以外からは付かなくなった。
  手書きの header リテラルが消え、付け忘れ・偽装値混入の面が縮小した。
- 認証ガードの制御フローが名前で読めるようになった（`requireMemberId` / `redirectIfAuthenticated`）。
  `checkout` だけ直呼びを残す線引きは本 ADR に固定する。
- ただしこれは付与点の集約であって、order 側の強制ではない。order は依然 `X-Member-Id` を無検証で
  信頼する（ADR-[[202606211100]]）。ヘッダを受理してよい呼び出し元を order / edge で検証する完全な
  強制点の確立は、引き続き将来の課題として残す。
- 新サーバを立てないため独立スケールは得られないが、学習用途では既存サーバ層を BFF と位置づける
  単純さの価値が高い。

## Alternatives considered

- **独立した BFF / API Gateway サーバを新設**: 入口を物理的に 1 つにできるが、store サーバ層が既に
  その役を担っており（ADR-[[202606170905]]）二重になる。Step 0 でのサーバ追加は早すぎる複雑化
  （ADR-[[202606170900]] のロードマップ）で却下。
- **`X-Member-Id` を `api` パッケージ（生成クライアント / mutator）に寄せる**: 付与点は 1 か所になるが、
  生成物 + baseURL 前置だけに保っている mutator に「検証済み memberId」という store 固有の関心が
  混ざる。生成クライアントの責務を逸脱するため却下。
- **memberId を AsyncLocalStorage で暗黙伝播し mutator が拾う**: 呼び出し側の引数が消えるが、伝播が
  暗黙になり追いにくく、薄く保つ方針に反する。却下。
- **`checkout` も `requireMemberId` で throw redirect に統一**: 制御フローは揃うが、action で Response を
  catch して error envelope に変換できず既存 UX（フォーム再表示）を壊す。却下。
