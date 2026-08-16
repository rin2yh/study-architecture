# ADR-202606170904: フロントエンドは TanStack Start（BFF を見据える）

- Status: Superseded by ADR-[[202606170908]]
- Date: 2026-06-17

## Context

UI は TypeScript + Vite でドメイン分割（store / mypage / backoffice）、出発点はブラウザ →
サービスの直接呼び出し。将来マイクロサービス化（ADR-[[202606170900]]）すると、ブラウザが多数
サービスへファンアウトする形は CORS・認証・集約で辛くなる。

## Decision

**TanStack Start + TanStack Query + orval(zod)** を採用する（UI 実装は後続イテレーション）。

- 決め手は、各 UI が持つ**サーバ層をそのまま BFF に育てられる**こと。「UI ごとのサーバ層」は
  将来コストではなく、集約・認証・段取りを置く継ぎ目になる。
- 多サービス呼び出しのキャッシュ・重複排除・リトライは TanStack Query が担う。
- クライアントと zod は OpenAPI から orval で生成する（ADR-[[202606170901]]）。

## Consequences

- Step 0 では各 UI に Node サーバが付き構成要素が増えるが、その分ファサード導入が別ゲートウェイを
  立てずに済む自然な拡張になる。

## Alternatives considered

- **TanStack Router + Query（純SPA）**: 現構成に最も素直だが BFF 層を持たず、
  マイクロサービス化時に集約層を別途用意する必要がある。
- **Remix v3**: 新世代だがエコシステム/安定度が新しめで、学習の主題（アーキテクチャ）に
  集中したい基盤プロジェクトには FW 自体の変動リスクが乗る。
