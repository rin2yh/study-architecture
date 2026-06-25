# ADR-202606220300: フロントエンドのコンポーネントを Feature-Sliced Design で層分けする

- Status: Accepted
- Date: 2026-06-22
- Relates to: ADR-[[202606170905]] / ADR-[[202606170908]] / ADR-[[202606190901]] / ADR-[[202606170906]]

## Context

`client/app/{store,mypage,backoffice}` の UI は、ドメインロジック・共有 UI キット (shadcn)・ユーティリティ・ルートが `src/` 直下にフラットに同居し、整理軸が React Router の `routes/` しかなく、層 (どこが再利用可能か) や依存方向の規律が無かった。さらに 1 つの `routes/*.tsx` に表示・入力フォーム・サーバ glue が詰め込まれ、再利用も差分レビューもしにくかった。3 つの UI は規模差が大きく (store はカート・チェックアウト、backoffice は一覧 1 枚)、一律のフォルダ規約では過不足が出る。

## Decision

整理軸として **Feature-Sliced Design (FSD)** を採用する。レイヤーは上位ほど具体・下位ほど汎用で、import は `app → pages → widgets → features → entities → shared` の一方向に流す。各スライスの公開境界は `index.ts` (Public API) に集約する。

- **実用本位で必要なレイヤー・スライスだけ作る** (空レイヤーは置かない)。規模差を一律規約で潰さず、backoffice / mypage は最小構成に留める。
- **`app` / `pages` 層は React Router (ADR-[[202606170908]]) の構成要素にそのまま割り当てる**。ルーティングは `routes.ts` で明示するため `routes/` 配下に置いても自動ルート化されず、別途 `src/pages/` を作って re-export する adapter は置かない。これにより「ページ」概念の二重化を避け、route の生成型 (`Route.ComponentProps` 等) をページが直接使える。
- route モジュールはファイル名を `route.tsx` で固定する (フォルダ名との重複回避)。ページ専用の表示コンポーネントは `routes/<page>/components/` にコロケートする。`routes/` はフレームワーク都合のページ層で FSD の正式スライスではないため、セグメント名は正式スライスの `ui` と区別して `components` とする。
- **操作 (mutation を起こす interaction) は `features`、純粋な表示はページにコロケート**を軸に切る。表示専用を `entities/*/ui` へ上げるのは複数ページで再利用が出た時点に留保する。API 取得は既存の `api` ワークスペースパッケージ (ADR-[[202606190901]]) を下層として使い `shared/api` は新設しない。サーバ側ローダ方針は ADR-[[202606170905]] を踏襲する。

## Consequences

- 「再利用可能か / ドメイン実体か / 操作か / 画面か」が配置から読め、依存方向が一方向に揃う。
- 詰め込み 1 ファイルが用途ごとに割れ、差分レビューと再利用がしやすくなった。
- `src/pages/` と adapter という二重のページ概念が無く、テストも route モジュールから default と `loader`/`action` を素直に import できる。
- vitest は tsconfig paths を読まないため各 app の `vitest.config.ts` に `@` の `resolve.alias` を追加した。
- トレードオフ: ページ UI が route モジュールと一体化し、フレームワーク非依存な「純粋 pages 層」にはならない。React Router を ADR で固定済みで可搬性の価値が低いため許容する。
- FSD のレイヤー間 import 規律は現状 lint で機械強制しておらず人手レビュー依存。自動化したくなれば steiger 等を別途検討する。

## Alternatives considered

- **フラット構成を維持**: 追加コストは無いが層・依存方向の規律が生まれず詰め込みも解消しない。
- **Atomic Design**: 見た目サイズで切るためドメインを跨いだ整理になり、ドメイン分割重視の本リポジトリ (ADR-[[202606170900]]) と相性が悪い。
- **正準フル FSD を全 app に強制**: 一貫性は最大だが一覧 1 枚の app まで全層通すと空スライスを量産する。「必要な分だけ作る」を採る。
- **独立した `src/pages/` 層 + `routes/*.tsx` を薄い adapter にする**: pages 層をフレームワーク非依存に保てるが「ページ」概念が二重化し、生成型が使えず手書き型に振り替えが要る。一度実装したが重複解消のため畳み直した。
