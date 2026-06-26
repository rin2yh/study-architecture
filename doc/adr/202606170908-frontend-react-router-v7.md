# ADR-202606170908: フロントエンドは React Router v7 (旧 Remix 統合) に切替

- Status: Accepted
- Date: 2026-06-17
- Supersedes: ADR-[[202606170904]]

## Context

ADR-[[202606170904]] で TanStack Start を採用したが、Docker 本番ビルドで Nitro が混在させる
Vite-dev 用 SSR fallback (`fetch(req, {viteEnv:"ssr"})`) によりブラウザ要求が
self-fetch デッドロックする問題が顕在化した。

回避策として薄い Node http サーバ (`client/start-server.mjs`) を自前で実装したが、
「SSR フレームワークを使っているのに `start-server` を自前で持つ必要がある」
状況は採用判断の前提を満たさない。

候補は次の通り。

- React Router v8 (pre-release のみ) / Remix v3 (npm 未公開、v7 に統合)
- React Router v7 (旧 Remix を取り込んだ安定版)
- SvelteKit / Next.js 等の他系統

## Decision

**React Router v7** (`react-router` / `@react-router/dev` / `@react-router/node` /
`@react-router/serve`) を採用する。

- Vite を build/dev で使い続ける (`@react-router/dev/vite` の `reactRouter()` plugin)。
- production 起動は `react-router-serve ./build/server/index.js` で完結 (自前 listen 不要)。
- ルートは `src/routes.ts` に明示的に書き、`Route.ComponentProps` / `Route.ErrorBoundaryProps`
  などの型は `react-router typegen` が生成。
- `loader` / `ErrorBoundary` / `HydrateFallback` を route ファイルに同居させる規約に従う。
- ADR-[[202606170905]] の「サーバ側ローダ + orval(zod)」は方針として維持 (実装は `loader` 内で
  `@ec/api` の生成クライアントを呼ぶ形)。

## Consequences

- `client/start-server.mjs` を廃止 (Vite-dev SSR 経路が Nitro 由来でなくなるため)。
- 各 app の `package.json` の `start` は `react-router-serve` を直接呼ぶ。
- 既存の `routes/index.tsx` / `__root.tsx` / `router.tsx` / `routeTree.gen.ts` を撤去し、
  `src/root.tsx` + `src/routes.ts` + `src/routes/home.tsx` 構成に書き換える。
- Docker (compose) で `localhost:5173/5174/5175` の SSR が安定動作することを確認済。
- BFF への育成方針 (ADR-[[202606170904]] の意図) は維持される (loader はサーバ側で実行、
  ブラウザはオリジンだけを叩く)。

## Alternatives considered

- **TanStack Start を維持して `nitro/vite` plugin の SSR rewrite を待つ**: 上流の
  挙動依存度が高く、解決時期が読めない。
- **CSR + Nitro routeRules proxy**: SSR を捨てる選択肢。マイクロサービス化時の
  BFF 育成方針 (Step 1) と相性が悪く却下。
- **React Router v8 (pre-release)**: 安定リリース未到来、本プロジェクトの学習主題が
  FW 安定性に依存するため却下。安定版が出たタイミングで再評価。
- **SvelteKit / Next.js**: React を捨てる / app router に乗り換える書き換え量が大きく、
  既存の React + Tailwind 構成を流用するコスト削減が効かない。
