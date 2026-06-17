# Known Issues

最終更新: 2026-06-17

## TanStack Start (Nitro 本番ビルド) の SSR self-request が Docker と相性悪い

- **症状**: `docker compose up` で UI を起動するとブラウザ要求に対し SSR がハング/500。
  - `localhost:5173/` → `AbortError`, `[::1]:5173/` → 500 (`fetch failed`)
  - コンテナ内から backend (`http://product:8080`) は 200 で取れる。
  - コンテナ内でローダを直接呼ぶと正常に商品データを返す（`process.env` も解決済み）。
- **原因の見立て**: TanStack Start (Nitro 3 beta) の本番ハンドラが SSR 時にホスト名/ポートで
  自己 fetch を行い、Docker (OrbStack) の network/loopback と相性が悪く失敗する。
- **dev サーバでは再現しない**: `pnpm --filter store dev` ＋ host から backend を直接 URL 指定
  すれば SSR で商品/注文/配送が正しく描画される（検証済み）。
- **影響範囲**: Docker compose での UI コンテナ運用のみ。バックエンド5サービスと migrate は
  Docker で完全動作。host 直接起動の dev SSR は全UI動作（ADR 0005/0006 のフローは成立）。
- **次の打ち手（候補）**:
  - Nitro の preset/server オプションで internal request handling を `node-server` 内に閉じる
  - Vite + TanStack Start のバージョンを安定版（1.0 到達後）に更新
  - SSR をオフにして CSR + TanStack Query に切替（ADR 0006 の見直し）
- **暫定回避**: 開発・検証は host の `pnpm dev` で行う。本番起動は本イシュー解消後に
  改めて Docker E2E を取る。

### 詳細調査 (2026-06-17)

- ハングの一次原因は `.output/server/index.mjs` 内に出力される `ssrRenderer`:
  ```js
  function ssrRenderer({ req }) {
    return fetch(req, { viteEnv: "ssr" });
  }
  ```
  これは Vite dev mode 用の SSR フォールバックで、production build にもそのまま残る。
  `req.url` (= `http://127.0.0.1:5173/`) に対して self HTTP fetch → 同一プロセスの
  Nitro handler が応答を返せず TCP 接続は accept されるが body が返らないデッドロックに
  なり、`AbortError`/`500 fetch failed` が出力される。
- `findRoute('/**')` は `_lazy_YuDFPc → ssr_renderer_exports` で固定登録されており、
  TanStack Start の `createStartHandler(defaultStreamHandler)` (`routes-*.mjs` に同梱)
  が Nitro ルーティングに反映されていない。
- 試したこと (いずれも改善なし):
  - `nitro/vite` plugin に `preset: "node-server"` を明示
  - `nitro` を `nitro-nightly@4.0.0-20251010` から `nitro@3.0.0` 安定版にダウングレード
  - `@tanstack/react-start` を 1.168.25 → 1.168.26 に更新 (相関する react-router 等含む)
- 結論: 現在の `nitro/vite` + `@tanstack/react-start/plugin/vite` の組み合わせでは
  production build における SSR エントリ rewrite が走らない可能性が高い。upstream の
  挙動を追う必要があるためこのまま open とする。
- 次の打ち手候補:
  - 上流の修正待ち / 1.169.x 以降に上げて再検証 (minimumReleaseAge 経過後)
  - `src/start.ts` で `createStart()` を明示し entry を制御する手順を追加
  - 最終手段として CSR + Nitro `routeRules` proxy に切替 (ADR 0006 改訂)
