# ADR-202606170908: フロントエンドは React Router v7 (旧 Remix 統合) に切替

- Status: Accepted
- Date: 2026-06-17
- Supersedes: ADR-[[202606170904]]

## Context

ADR-[[202606170904]] で TanStack Start を採用したが、Docker 本番ビルドで Nitro が混在させる
Vite-dev 用 SSR fallback によりブラウザ要求が self-fetch デッドロックした。

回避策として薄い Node http サーバを自前で実装したが、「SSR フレームワークを使っているのに
start-server を自前で持つ」状況は採用判断の前提を満たさない。

## Decision

**React Router v7**（旧 Remix を取り込んだ安定版）を採用する。

- 決め手は、production 起動が `react-router-serve` で完結し**自前 listen が要らない**こと。
  build/dev は Vite plugin で従来どおり。
- ルートは明示的に列挙し、route の型は `react-router typegen` が生成する。
- ADR-[[202606170905]] の「サーバ側ローダ + orval(zod)」は方針として維持する。

## Consequences

- 自前の start-server を廃止し、各 app の `start` は `react-router-serve` を直接呼ぶ。
- TanStack Router 由来のルート定義ファイル群を React Router v7 の構成へ書き換える。
- BFF への育成方針（ADR-[[202606170904]] の意図）は維持される（loader はサーバ側で実行、
  ブラウザはオリジンだけを叩く）。

## Alternatives considered

- **TanStack Start を維持して上流の SSR rewrite 修正を待つ**: 上流の挙動依存度が高く、解決時期が
  読めない。
- **CSR + Nitro routeRules proxy**: SSR を捨てる選択肢。BFF 育成方針と相性が悪く却下。
- **React Router v8 (pre-release)**: 安定リリース未到来。学習の主題を FW 安定性に依存させないため却下。
  安定版が出たタイミングで再評価。
- **SvelteKit / Next.js**: React を捨てる / app router に乗り換える書き換え量が大きく、既存の
  React + Tailwind 構成を流用するコスト削減が効かない。
