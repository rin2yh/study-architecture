# ADR-202606170905: UI のデータ取得はサーバ側ローダ + orval(zod)

- Status: Accepted (FW は ADR-[[202606170904]] → ADR-[[202606170908]] により React Router v7 へ置換、本 ADR の方針 (サーバ側ローダ + orval/zod) は維持)
- Date: 2026-06-17

## Context

ADR-[[202606170904]] で UI を TanStack Start に決めた。Step 0 の「直接呼び出し（別ファサードを立てない）」を
保ちつつ、ブラウザ→各サービスの直接ファンアウトに伴う CORS・認証・集約の負担を避けたい。
各 Go サービスには現状 CORS ミドルウェアが無い。

## Decision

UI のデータ取得は **TanStack Start のサーバ側（ローダ / `createServerFn`）から各サービスを
HTTP 呼び出しする**。生成クライアントは **orval**（`client: 'fetch'` + zod スキーマ）。

- データフロー: ブラウザ → UI(:3000) の SSR/サーバ関数 → `http://<svc>:8080`（compose 内部DNS）
  → Go サービス。ブラウザは UI のオリジンだけを叩くため **CORS 不要**。
- サービス URL は **サーバ側 env**（`PRODUCT_API_URL` 等）で注入。orval の `override.mutator`
  でサービスごとの fetcher（`productFetch` 等）が `process.env` から baseURL を前置する。
- レスポンスは orval 生成の **zod スキーマで検証**してから利用（`ListProductsResponse.parse`）。
- 生成コード（`client/package/api/src/**`）は共有パッケージ `@ec/api` に集約してコミットし、
  Docker ビルドは `client/` ワークスペースに閉じる（生成は `pnpm api:gen`、入力は
  `server/<svc>/api/openapi.yaml`）。詳細は ADR-[[202606170906]]。

「別ファサードを立てない」という Step 0 方針は、UI 自身のサーバ層が呼ぶ形なので維持される。
この UI サーバ層が ADR-[[202606170904]] の言う Step 1 の BFF/ファサードへ自然に育つ。

## Consequences

- Go サービスに CORS を入れずに済む（サービスは内部向けのまま保てる）。
- サービス URL・将来のトークンがクライアントへ漏れない（サーバ側に閉じる）。
- クライアント側の対話的データ取得（カート操作等）が必要になった時点で、orval の
  `react-query` ターゲット（`useQuery`/`useMutation`）を追加し、公開URL経路を足す。
  その際 baseURL のサーバ/クライアント切替は `createIsomorphicFn` で行う。

## Alternatives considered

- **ブラウザから各サービスを直接 fetch + 各サービスに CORS 追加**: Step 0 の直接呼び出しに
  最も忠実だが、5 サービス全てに CORS を入れる必要があり、認証・集約も各所に分散する。
- **別途 API ゲートウェイ/ファサードサービスを新設**: Step 0 の「ファサードなし」に反し、
  早すぎる複雑化（ロードマップ上は Step 1）。
