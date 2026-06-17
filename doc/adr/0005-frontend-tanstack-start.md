# ADR 0005: フロントエンドは TanStack Start（BFF を見据える）

- Status: Accepted
- Date: 2026-06-17

## Context

UI は TypeScript + Vite、ドメイン分割（store / mypage / backoffice）。バリデーションは zod。
出発点はブラウザ → サービスの直接呼び出し。一方で将来マイクロサービス化（[[0001]] の
Step 1〜3）した際、ブラウザが多数サービスへファンアウトする形は CORS・認証・集約で辛くなる。

候補は Remix v3 / TanStack Start（フルスタック）/ TanStack Router+Query（純SPA）。

## Decision

**TanStack Start + TanStack Query + orval(zod)** を採用する（UI 実装は後続イテレーション）。

- 各 UI が持つ**サーバ層をそのまま BFF（Backend-For-Frontend）= Step 1 のファサード**に育てる。
  「UI ごとのサーバ層」は将来コストではなく、集約・認証・段取りを置く継ぎ目になる。
- **TanStack Query** が多サービス呼び出しのキャッシュ・重複排除・リトライを担う。
- Vite ネイティブ・型安全ルーティング・server functions で「Vite + TS」方針に素直に乗る。
- orval で OpenAPI → fetch クライアント + zod スキーマを生成（[[0002]]）。

## Consequences

- Step 0 では各 UI に Node サーバが付き構成要素が増えるが、Step 1 のファサード導入が
  自然な拡張になる（別途ゲートウェイを立てずに済む）。
- バックエンド各サービスの OpenAPI から orval で型・クライアント・zod を生成し、
  フロントの手書きを減らす。

## Alternatives considered

- **TanStack Router + Query（純SPA）**: 現構成に最も素直だが BFF 層を持たず、
  マイクロサービス化時に集約層を別途用意する必要がある。
- **Remix v3**: 新世代だがエコシステム/安定度が新しめで、学習の主題（アーキテクチャ）に
  集中したい基盤プロジェクトには FW 自体の変動リスクが乗る。採用前に現状成熟度の
  一次情報確認を要する（UI 着手時に再評価）。
