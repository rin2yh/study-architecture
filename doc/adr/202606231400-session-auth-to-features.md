# ADR-202606231400: 認証ロジックを entity ではなく features/auth へ集約する

- Status: Accepted
- Date: 2026-06-23
- Supersedes: ADR-[[202606230930]] (認証ガードの配置先 `entities/session` の部分のみ)
- Relates to: ADR-[[202606220300]] / ADR-[[202606211100]]

## Context

FSD レイヤリング (ADR-[[202606220300]]) で `store` の `entities/` は `cart` と `session` を持つ。
レビューで「entities が薄い・細かい」と挙がり、`session` の配置を再評価した。当初「`shared/lib`
へ移す」案を検討したが誤りと判明し、最終的に **`session` は entity ではなく `features/auth` の
一部** という結論に至った。

判断の軸を「**クライアント側のドメインデータモデルを所有するか**」に置いて計測すると、`cart` と
`session` は本質的に異なる:

- `cart`: `CartItem[]` を **localStorage に状態として所有**し (`store.cart.v1`)、複数フィーチャ
  (add-to-cart / checkout) がその状態を読み書きする。→ データを持つ **entity**。
- `session`: ADR-[[202606211100]] によりセッションは **サーバ側**。クライアントが持つのは中身を
  読めない HttpOnly Cookie だけで、`currentMemberId` は毎回 member API を引く **ラウンドトリップ**。
  **クライアント側に所有するデータモデルが無い**。中身は「現在会員の解決 / 認可ガード / Cookie の
  発行・失効」という **振る舞い (verb) のみ**。→ entity ではなく **feature**。

`shared/lib` 案が誤りだった理由: Cookie ヘルパ (`SESSION_COOKIE` / `readSessionToken` /
`sessionCookie` / `clearSessionCookie`) は `member_session` という固有名と認証用 Max-Age を知る
**session 固有**の実装であり、ドメイン非依存ではない。`cart` の localStorage が `store.cart.v1`
固有で `entities/cart` に凝集しているのと同じく、`shared` (真にドメイン非依存な `money` 等の置き場)
には出せない。

## Decision

`session.ts` を丸ごと **`features/auth/model/session.ts`** へ移す。`features/auth` は既に `ui/`
(`LoginForm` / `LogoutButton`) を持つので、認証の振る舞い一式をこのスライスに凝集させる。Cookie
ヘルパ・`currentMemberId`・ガード (`requireMemberId` / `redirectIfAuthenticated`) はいずれも auth
固有なので分割せず同居させる。公開 API は `features/auth/index.ts` バレルに集約し、`routes/` は
バレル経由で消費する (`entities/cart` の消費規約と同型)。

レイヤの判断軸を次に固定する:

- **`entities/`** — クライアント側のドメインデータモデルを所有する名詞 (例: `cart` の `CartItem[]`)。
- **`features/`** — データモデルを所有しない、ユーザ価値のある振る舞い・操作 (例: `auth` の
  ログイン / ログアウト / 認可ガード / 現在会員の解決)。
- **`shared/lib/`** — 真にドメイン非依存なユーティリティのみ (例: `money` の `yen` 整形)。

結果として `entities/` は唯一の真のデータ実体である `cart` のみになる。「entities が薄い」という当初
の違和感の正体は、**データを持たない振る舞いの塊を entity に置いていたこと**で、本来 feature だった。

`requireMemberId` / `redirectIfAuthenticated` は引き続き認証集約の役割を担う (ADR-[[202606230930]]
の意図は不変)。変わるのは**配置層だけ** (`entities/session` → `features/auth/model`)。

## Consequences

- `entities/` が `cart` だけになり、entity = クライアントデータを持つ実体、という軸が配置から読める。
- 認証の転送・解決・ガード・UI が `features/auth` に凝集し、参照は公開バレル経由の一方向 (routes →
  features/auth → api/member・shared)。`shared` を経由しないため `shared` のドメイン非依存性は保たれる。
- ADR-[[202606230930]] の「loader 用ガードを `entities/session` に置く」は配置層のみ
  `features/auth/model` に更新 (集約の挙動・X-Member-Id 付与点は不変)。
- ADR-[[202606220300]] の entities 例から `session` が外れ `cart` のみが残る。`session` は features の例に移る。
- カバレッジは `src/features/**` の既存 glob が拾うため、vitest.config.ts の個別パス指定を削除。
- 同じ軸で `entities/cart` を見直し、checkout 専用の射影 `toCheckoutItems` (cart→注文 API 形状,
  ADR-[[202606190900]]) を `features/checkout` へ移した。`entities/cart` は cart 自身のモデル
  (`CartItem` + 操作 + `useCart`) のみを持つ。

## Alternatives considered

- **`entities/session` のまま据え置き (ADR-[[202606230930]])**: FSD では session を entity 例に
  挙げる文脈もあるが、それは**クライアント側に session データモデルを持つ**前提。本アプリはサーバ側
  セッション (ADR-[[202606211100]]) でクライアントは振る舞いしか持たないため entity の要件を満たさない。不採用。
- **`shared/lib/session` へ移す (本 PR の初版)**: Cookie ヘルパが `member_session` 固有でドメイン
  非依存でなく、`currentMemberId` は member ドメインに依存する。`shared` 汚染になるため不採用。
- **infra (Cookie) / domain (identity・ガード) で 2 スライスに割る**: 1 つの auth 関心が複数の置き場
  に散り、`cart` が永続化も操作も `entities/cart` に凝集しているのと非対称になる。凝集を優先して不採用。
