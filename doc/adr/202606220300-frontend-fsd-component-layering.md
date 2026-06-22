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

`app → pages → widgets → features → entities → shared`

実用本位で運用し、**必要なレイヤー・スライスだけを作る** (空レイヤーは置かない)。`widgets` は
合成ブロックが増えた箇所でだけ起こす。各スライスの公開境界は `index.ts` (Public API) に集約し、
外からは barrel 経由で参照する。

**`app` / `pages` 層は React Router (ADR-[[202606170908]]) の構成要素にそのまま割り当てる**。
本リポジトリのルーティングは `routes.ts` で明示するため (ファイルベースではない)、`routes/`
配下に route 以外のファイルを置いてもルート化されない。これを利用し、別に `src/pages/` を作って
re-export する adapter は**置かない**:

- `app` 層 = `root.tsx` (html shell / `Outlet` / グローバル ErrorBoundary) + `routes.ts` + `styles.css`。
- `pages` 層 = `routes/`。**route モジュールがページそのもの**で、`loader` / `action` /
  default / `ErrorBoundary` / `HydrateFallback` を持つ (フレームワーク契約はページの責務)。
  そのページ専用の表示コンポーネントは `routes/<page>/components/` に**コロケート**する
  (例: `routes/home/components/product-row.tsx`、
  `routes/cart/components/{cart-list,cart-row,empty-cart}.tsx`)。`routes/` はフレームワーク
  都合のページ層で FSD の正式スライスではないため、セグメント名は `ui` ではなく `components`
  とする (正式スライスの `features/*/ui`・`entities/*/ui` は FSD 規約どおり `ui` のまま)。

サーバ側ローダの方針は ADR-[[202606170905]] を踏襲する。

下位レイヤーの配置:

- `shared/` — ドメイン非依存の再利用部品。`shared/ui/*` (shadcn の UI キット +
  `page-loading`)、`shared/lib/*` (`utils` の `cn` / `money` の `yen`)。
- `entities/` — ビジネス実体。`entities/cart` (モデル + `useCart` フック)、
  `entities/session` (Cookie セッション、ADR-[[202606211100]])。
- `features/` — ユーザー操作 (interaction)。`features/add-to-cart`、`features/checkout`
  (フォーム + `parseItems` + 結果型)、mypage の `features/auth` (`LoginForm` / `LogoutButton`)。

操作 (mutation を起こす interaction) は `features`、純粋な表示はページにコロケート、という軸で
切る。表示専用コンポーネントを `entities/*/ui` まで上げるのは、複数ページで再利用が出た時点に
留保する (現状はページ固有のためコロケート)。API 取得クライアントは既存の `api` ワークスペース
パッケージ (ADR-[[202606190901]]) をそのまま下層として使い、`shared/api` は新設しない。

## Consequences

- 関心が層で分かれ、「再利用可能か (shared)」「ドメイン実体か (entities)」「操作か
  (features)」「画面か (pages = routes)」が配置から読める。依存方向が一方向に揃う。
- 詰め込み 1 ファイルが用途ごとのファイルに割れ、差分レビューと再利用がしやすくなった。
  例: `CheckoutForm` (feature) と `OrderConfirmed` / `EmptyCheckout` (ページにコロケート) が
  分離した。
- `routes/` を pages 層そのものに割り当てたので、`src/pages/` と adapter という**二重のページ
  概念が無い**。route モジュールは `Route.ComponentProps` / `Route.ActionArgs` 等の生成型を
  直接使え、テストも `./home` から default と `loader`/`action` を素直に import できる。
- vitest は tsconfig paths を読まないため、各 app の `vitest.config.ts` に `@` の
  `resolve.alias` を追加した (mypage は今回初めて `@/` を使うため)。coverage の `include` も
  `src/routes/**` 中心へ更新した。
- トレードオフ: ページ UI がフレームワークの route モジュールと一体化する (フレームワーク
  非依存な「純粋 pages 層」にはならない)。本リポジトリは React Router 採用を ADR で固定済みで
  可搬性の価値が低いため許容する。小規模 app では層を増やしすぎないよう、backoffice
  (テーブル 1 枚) と mypage は最小構成に留め、無理に `entities`/`features`/`widgets` を切らない。
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
- **独立した `src/pages/` 層 + `routes/*.tsx` を薄い adapter にする**: pages 層をフレームワーク
  非依存に保てるが、`routes/` と `src/pages/` の両方が「ページ」を表し概念が二重化する。ルート
  ごとに re-export adapter と barrel が増え、route の生成型 (`Route.ComponentProps` 等) も
  ページ側で使えず手書き型に振り替える必要が出る。本リポジトリのルーティングは明示設定で
  コロケートが安全なため、`routes/` を pages 層に直接割り当てる方を採り却下
  (この案で一度実装したが、重複を解消するため畳み直した)。
