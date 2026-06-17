# Known Issues

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
