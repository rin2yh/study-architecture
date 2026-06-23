# ADR-202606231400: session モジュールを entities から shared/lib へ移す

- Status: Accepted
- Date: 2026-06-23
- Supersedes: ADR-[[202606230930]] (認証ガードの配置先 `entities/session` の部分のみ)
- Relates to: ADR-[[202606220300]] / ADR-[[202606211100]]

## Context

FSD レイヤリング (ADR-[[202606220300]]) で `store` の `entities/` は `cart` と `session` の
2 スライスを持つ。レビューで「entities が薄い・細かい印象」が挙がり、ADR-[[202606220300]]
自身の「必要なスライスだけ作る・無理に切らない」方針に照らして `session` の配置を再評価した。

計測すると、両スライスの性質が異なる:

- `cart` (97 行): `CartItem` 型 + 操作群 (add/remove/数量/合計/checkout 変換) + `useCart` フック。
  **固有の型と振る舞いを持つドメイン実体**で、3 箇所 (add-to-cart / cart / checkout) から再利用される。
- `session` (55 行): 大半が **横断的な転送配線** (Cookie の読み書き・失効)。残りは認証コンテキスト
  ヘルパ (`currentMemberId` / `requireMemberId` / `redirectIfAuthenticated`, ADR-[[202606230930]])。
  固有のドメインモデル (session 集約の型) は持たず、`shared/lib/money` (`yen` 整形) と同じ
  **モデルを持たない横断ユーティリティ**の性格が強い。

ADR-[[202606230930]] は loader 用認証ガードを `entities/session` に置くと決めたが、これは
「認証コンテキストを 1 か所へ集約する」判断であって、その置き場が entities である必然は無い。

## Decision

`session` モジュールを `entities/session/model/session.ts` から **`shared/lib/session.ts`** へ移す
(`shared/lib/money` と同じくバレル無しのフラット配置)。移動後 `entities/` は `cart` 1 スライスだけになる。

判断軸を次に固定する:

- **`entities/`** — 固有のドメインモデル (型 + 振る舞い) を持ち、複数 features/pages から再利用される
  実体だけを置く (例: `cart`)。
- **`shared/lib/`** — モデルを持たない横断ユーティリティ (例: `money` の整形、`session` の Cookie
  転送 + 認証ガード) を置く。

`requireMemberId` / `redirectIfAuthenticated` は引き続き認証集約の役割を担う (ADR-[[202606230930]]
の意図は不変)。変わるのは**置き場だけ**。将来 session が固有のモデルを持つほど厚くなった時点で
`entities/session` へ昇格させる。

## Consequences

- `entities/` が `cart` だけになり、「薄い entity が並ぶ」印象が解消する。entities = 厚いドメイン
  実体、という軸が配置から読める。
- **トレードオフ (domain 結合)**: `shared/lib/session` は `api/member` の `getSession` と
  react-router の `redirect` に依存するため、`shared` が厳密なドメイン非依存ではなくなる。生成
  `api` パッケージは ADR-[[202606220300]] が下層依存として許容しており、import 方向 (routes →
  shared) は一方向のまま壊れない。許容できなくなれば `entities/session` へ戻す。
- ADR-[[202606230930]] の「loader 用ガードを `entities/session` に置く」は配置先のみ
  `shared/lib/session` に更新 (集約の挙動・X-Member-Id 付与点は不変)。
- ADR-[[202606220300]] の entities 例から `session` が外れ、`cart` のみが残る。
- カバレッジ対象パスを `src/shared/lib/session.ts` に更新 (vitest.config.ts)。

## Alternatives considered

- **`entities/session` のまま据え置き (現状 / ADR-[[202606230930]])**: FSD の定石では session は
  代表的な entity だが、モデルを持たない薄いスライスが entities に残りレビュー指摘が解消しない。
  entities を厚いドメインモデル専用に保つため不採用。
- **session を分割 (Cookie 転送 → shared、認証ガード → entities)**: 1 つの関心が 2 か所に割れて
  粒度が**増える**。「薄いうちは分けない」というレビューの意図と逆行するため不採用。
- **厚くなるまで移動を保留**: 筋は通るが、薄さは現状すでにあり、レビューは今の是正を求めている。
  今移動し、`entities/session` へ戻す昇格条件 (固有モデルの獲得) を本 ADR に明記する形を採った。
