# ADR-202606230930: store BFF に認証コンテキストの集約と X-Member-Id の付与点を置く

- Status: Accepted (認証ガードの配置先 `entities/session` は ADR-[[202606231400]] で `features/auth/model` に更新)
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

スコープは store のみ。新サーバ・新 `shared/api`・新スライスは作らない（`shared/api` 新設は
ADR-[[202606220300]] が否定。生成クライアントの `api` パッケージはその下層として直接使う）。

1. **認証コンテキストの集約**: loader 用ガードを `entities/session` に置く。`requireMemberId(request)`
   は未認証なら `throw redirect("/login")`、認証済みなら `memberId` を返す。`login` の逆向きガードは
   `redirectIfAuthenticated(request)`。`checkout` の action は redirect ではなく error envelope を返す
   別仕様で、`throw redirect` を action 内で catch すると Response を壊すため、ここはガードに寄せず
   `currentMemberId` の直呼びを残す。throw / return の 2 モードを引数で切り替えるとガードが太るので、
   1 関数 1 責務に保つ。

2. **X-Member-Id の付与点を loader に閉じる**: `orders` の loader で `requireMemberId` が返す検証済み
   `memberId: number` からのみ `X-Member-Id` を付けて `listOrders` を呼ぶ。付与のためだけの専用スライス
   （`entities/order`）やラッパは作らない。呼び出し元が `orders` 1 箇所で、route が `api/order` を
   直呼びするのは `checkout`（`import { checkout } from "api/order"`）と揃った既存パターンであり、3 行の
   ラッパ 1 本だけの空スライスは ADR-[[202606220300]] の「必要なスライスだけ・空スライスは置かない」に
   反するため。ブラウザ由来の値が乗らない保証は、付与の引数を `requireMemberId` のサーバ検証済み戻り値に
   限ることで担保する。将来 order の他エンドポイントや他サービスも `X-Member-Id` を要求しはじめ、付与点が
   複数に散り始めた時点で、共通の付与ヘルパを `entities/session` か小さな共有 lib へ括り出す。

段取り（checkout の手順など）は order サービスに残し、BFF 層は認証集約に限って薄く保つ
（ADR-[[202606170900]] Step 1）。

## Consequences

- `X-Member-Id` の付与点が `orders` の loader 1 箇所に閉じ、`requireMemberId` のサーバ検証済み戻り値
  からのみ付くようになった。ブラウザ由来の値が乗る経路は無い。
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
- **付与専用の `entities/order` スライス（`listMyOrders` ラッパ）を作る**: 付与点に名前は付くが、
  `listOrders` を呼ぶだけの 3 行ラッパ 1 本でモデルもロジックも持たず、`checkout` が `api/order` を
  直呼びするのと非対称。呼び出し元 1 箇所では空スライスの新設（ADR-[[202606220300]]）に当たり過剰なため
  却下し、loader にインライン化した。
- **memberId を AsyncLocalStorage で暗黙伝播し mutator が拾う**: 呼び出し側の引数が消えるが、伝播が
  暗黙になり追いにくく、薄く保つ方針に反する。却下。
- **`checkout` も `requireMemberId` で throw redirect に統一**: 制御フローは揃うが、action で Response を
  catch して error envelope に変換できず既存 UX（フォーム再表示）を壊す。却下。
