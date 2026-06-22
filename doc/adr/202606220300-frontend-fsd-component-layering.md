# ADR-202606220300: フロントエンドのコンポーネントを Feature-Sliced Design で層分けする

- Status: Accepted
- Date: 2026-06-22
- Relates to: ADR-[[202606170905]] / ADR-[[202606170908]] / ADR-[[202606190901]] / ADR-[[202606170906]]

## Context

`client/app/{store,mypage,backoffice}` の各 UI は、`src/` 直下にドメインロジック
(`cart.ts` / `session.ts`)、共有 UI キット (`components/ui/*` の shadcn)、ユーティリティ
(`lib/utils.ts` / `money.ts`)、ルートがフラットに同居していた。整理の軸が「React Router の
`routes/`」しかなく、層 (どこが再利用可能でどこがドメイン依存か) や依存方向の規律が無かった。

加えて 1 つの `routes/*.tsx` に複数コンポーネントが詰め込まれていた。例えば store の
`routes/home.tsx` は `ProductRow` / `Home` / `ErrorBoundary` / `HydrateFallback` の 4 つ、
`routes/checkout.tsx` は `OrderConfirmed` / `EmptyCheckout` / `CheckoutForm` / `parseItems` /
`action` / 画面本体を 1 ファイルに抱えていた。表示・入力フォーム・サーバ glue が混ざり、
再利用も差分レビューもしにくい。

3 つの UI は規模差が大きい。store はカート・チェックアウト・商品一覧と状態を持つ一方、
backoffice は一覧テーブル 1 枚、mypage はその中間である。一律のフォルダ規約だと store では
粗すぎ、backoffice では過剰になる。

## Decision

コンポーネントの整理軸として **Feature-Sliced Design (FSD)** を採用する。レイヤーは上位ほど
具体・下位ほど汎用で、import は上位→下位の一方向に流す:

`pages → widgets → features → entities → shared`

実用本位で運用し、**必要なレイヤー・スライスだけを作る** (空レイヤーは置かない)。`widgets` は
合成ブロックが増えた箇所でだけ起こす。各スライスの公開境界は `index.ts` (Public API) に集約し、
外からは barrel 経由で参照する。

React Router (ADR-[[202606170908]]) のファイルベースルーティングは `routes.ts` →
`routes/*.tsx` が `loader` / `action` を export する制約があるため、**`routes/*.tsx` は薄い
adapter** に留める。`loader` / `action` などフレームワーク glue だけを残し、画面 UI は `pages/`
から `export { ... as default }` で re-export する。サーバ側ローダの方針は ADR-[[202606170905]]
を踏襲する。

層ごとの配置:

- `shared/` — ドメイン非依存の再利用部品。`shared/ui/*` (shadcn の UI キット +
  `page-loading`)、`shared/lib/*` (`utils` の `cn` / `money` の `yen`)。
- `entities/` — ビジネス実体。`entities/cart` (モデル + `useCart` フック)、
  `entities/session` (Cookie セッション、ADR-[[202606211100]])。
- `features/` — ユーザー操作。`features/add-to-cart`、`features/checkout` (フォーム +
  `parseItems` + 結果型)、mypage の `features/auth` (`LoginForm` / `LogoutButton`)。
- `pages/` — ルートが描く画面の合成。詰め込みすぎだった `routes/*.tsx` をここで
  画面・テーブル・行・エラー境界・フォールバックに分割する。

API 取得クライアントは既存の `api` ワークスペースパッケージ (ADR-[[202606190901]]) を
そのまま下層として使い、`shared/api` は新設しない。

## Consequences

- 関心が層で分かれ、「再利用可能か (shared)」「ドメイン実体か (entities)」「操作か
  (features)」「画面合成か (pages)」が配置から読める。依存方向が一方向に揃う。
- 詰め込み 1 ファイルが用途ごとのファイルに割れ、差分レビューと再利用がしやすくなった。
  例: `CheckoutForm` (feature) と `OrderConfirmed` / `EmptyCheckout` (page) が分離した。
- `routes/*.tsx` が薄くなり、フレームワーク依存 (loader/action 型) と画面実装が切れた。
  テストは UI を `routes/*` 経由 (re-export) と直接の双方から従来どおり叩ける。
- vitest は tsconfig paths を読まないため、各 app の `vitest.config.ts` に `@` の
  `resolve.alias` を追加した (mypage / backoffice は今回初めて `@/` を使うため)。coverage の
  `include` も新パスへ更新した。
- トレードオフ: barrel (`index.ts`) と薄い adapter のぶんファイル数とボイラープレートが増える。
  小規模 app では過剰になりうるため、backoffice (テーブル 1 枚) と mypage は `pages` 中心の
  最小構成に留め、無理に `entities`/`features`/`widgets` を切らない。
- FSD のレイヤー間 import 規律は現状 lint で機械強制していない (人手レビュー依存)。違反検出を
  自動化したくなったら steiger 等の専用リンタ導入を別途検討する。

## Alternatives considered

- **現状のフラット構成を維持**: 追加コストは無いが、整理軸が `routes/` だけのままで層・依存方向の
  規律が生まれず、詰め込みファイルも解消しない。本 ADR の動機を満たさないため却下。
- **Atomic Design (atoms/molecules/organisms/…)**: 粒度の軸はあるが、UI の見た目サイズで切るため
  ドメイン (cart / order / session) を跨いだ整理になり、ビジネスロジックの置き場が曖昧になる。
  ドメイン分割を重視する本リポジトリ (サービスベース、ADR-[[202606170900]]) と相性が悪く却下。
- **正準フル FSD を全 app に強制 (shared/entities/features/widgets/pages を必ず全層)**: 一貫性は
  最大だが、backoffice のような一覧 1 枚の app まで全層を通すのは空スライスを量産するだけで
  過剰。「必要な分だけ作る」実用運用を採り、フル強制は却下。
