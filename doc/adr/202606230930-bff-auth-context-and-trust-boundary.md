# ADR-202606230930: store BFF に認証コンテキストの集約と X-Member-Id の付与点を置く

- Status: Accepted
- Date: 2026-06-23
- Relates to: ADR-[[202606170900]] / ADR-[[202606170905]] / ADR-[[202606211100]] / ADR-[[202606220300]]

## Context

ADR-[[202606170900]] Step 1（API ファサードで UI 入口を集約）に着手したところ、ファサード相当は既に
store のサーバ層（loader / action）に実装済みだった。Cookie セッション検証（ADR-[[202606211100]]）と
サーバ側に閉じたサービス呼び出し（ADR-[[202606170905]]）が揃い、ADR-[[202606170905]] / ADR-[[202606211100]]
ともこのサーバ層を Step 1 の BFF とし信頼境界を寄せる前提だった。よって新サーバを立てず既存サーバ層を
formalize する。現状の綻び:

- 認証チェックが route ごとに散在し、制御フローも不統一（redirect する route / error envelope を返す route が混在）。
- `X-Member-Id` の付与が `orders` の loader に手書きで、付与点が定まらず検証していない値の混入を防ぐ構造がない。

## Decision

スコープは store のみ。新サーバ・新 `shared/api`・新スライスは作らない（`shared/api` 新設は ADR-[[202606220300]]
が否定。生成クライアントの `api` は下層として直接使う）。

1. **認証コンテキストの集約**: loader 用ガードを `entities/session` に置く（未認証なら redirect する
   `requireMemberId`、既認証を弾く `redirectIfAuthenticated`）。`checkout` の action だけは error envelope を
   返す仕様で、action 内で `throw redirect` を catch すると Response を壊すため `currentMemberId` 直呼びを残す。
   why: throw / return を引数で切り替えるとガードが太るので 1 関数 1 責務に保つ。

2. **X-Member-Id の付与点を loader に閉じる**: `requireMemberId` が返す検証済み `memberId` からのみ付与する。
   why: ブラウザ由来の値が乗らない保証を「付与の引数をサーバ検証済み戻り値に限る」ことで担保する。専用スライス
   （`entities/order`）は作らず loader にインライン化する（3 行ラッパの空スライスは ADR-[[202606220300]] に反する）。
   付与点が複数に散り始めた時点で共通ヘルパへ括り出す。

段取り（checkout の手順など）は order サービスに残し、BFF 層は認証集約に限って薄く保つ（ADR-[[202606170900]] Step 1）。

## Consequences

- `X-Member-Id` の付与点が loader 1 箇所に閉じ、ブラウザ由来の値が乗る経路が無くなった。認証ガードの制御フローが
  名前で読める。`checkout` だけ直呼びを残す線引きは本 ADR に固定する。
- これは付与点の集約であって order 側の強制ではない。order は依然 `X-Member-Id` を無検証で信頼する
  （ADR-[[202606211100]]）。受理してよい呼び出し元を order / edge で検証する完全な強制点は将来の課題。
- 新サーバを立てないため独立スケールは得られないが、学習用途では既存サーバ層を BFF と位置づける単純さを優先した。

## Alternatives considered

- **独立した BFF / API Gateway サーバを新設**: store サーバ層が既に役を担っており（ADR-[[202606170905]]）二重になる。
  Step 0 でのサーバ追加は早すぎる複雑化（ADR-[[202606170900]]）で却下。
- **`X-Member-Id` を `api` パッケージ（mutator）に寄せる**: 付与点は 1 か所になるが、baseURL 前置だけに保つ
  生成クライアントの mutator に「検証済み memberId」という store 固有の関心が混ざり責務を逸脱するため却下。
- **付与専用の `entities/order` スライスを作る**: `listOrders` を呼ぶだけの 3 行ラッパで、`checkout` の直呼びと非対称。
  呼び出し元 1 箇所では空スライスの新設（ADR-[[202606220300]]）に当たり過剰なため却下。
- **memberId を AsyncLocalStorage で暗黙伝播**: 伝播が暗黙で追いにくく、薄く保つ方針に反するため却下。
- **`checkout` も `requireMemberId` で throw redirect に統一**: action で Response を catch して error envelope に
  変換できず既存 UX（フォーム再表示）を壊すため却下。
