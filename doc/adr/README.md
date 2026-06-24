# ADR 一覧

設計判断の記録。ID はファイル名先頭の作成日時タイムスタンプ (`YYYYMMDDHHmm`)。採番規約は
[ADR-202606211000](202606211000-adr-timestamp-naming.md)。相互参照は本文中で `ADR-[[<ID>]]` 形式で張る。

| ID | Status | タイトル |
| --- | --- | --- |
| [202606170900](202606170900-service-based-architecture.md) | Accepted | サービスベースアーキテクチャを採用する |
| [202606170901](202606170901-codegen-first-tech-stack.md) | Accepted | コード生成中心の技術スタック |
| [202606170902](202606170902-single-root-gomod-monorepo.md) | Accepted | 単一ルート go.mod のモノレポ構成 |
| [202606170903](202606170903-shared-postgres-schema-per-domain.md) | Accepted | 共有 Postgres + ドメインごとの schema 分離 |
| [202606170904](202606170904-frontend-tanstack-start.md) | Superseded | フロントエンドは TanStack Start（BFF を見据える） |
| [202606170905](202606170905-ui-server-loader-data-fetching.md) | Accepted | UI のデータ取得はサーバ側ローダ + orval(zod) |
| [202606170906](202606170906-frontend-pnpm-monorepo-tooling.md) | Accepted | フロントエンドは pnpm モノレポ + oxlint/oxfmt、命名は client/server・単数 |
| [202606170907](202606170907-go-web-framework-gin.md) | Accepted | Go サーバの HTTP フレームワークに Gin を採用 |
| [202606170908](202606170908-frontend-react-router-v7.md) | Accepted | フロントエンドは React Router v7 (旧 Remix 統合) に切替 |
| [202606170909](202606170909-split-customer-and-ops-db.md) | Accepted | 顧客系 (社外) と運用系 (社内) で DB とネットワーク経路を分離 |
| [202606180900](202606180900-migration-per-service.md) | Accepted | マイグレーションをサービスごとに分割する |
| [202606180901](202606180901-api-error-model.md) | Accepted | API エラーモデルを共通 Error スキーマ + ErrorJSON ミドルウェアに集約する |
| [202606180902](202606180902-repository-real-db-integration-test.md) | Accepted | repository 層は実 DB 結合テストで検証する |
| [202606180903](202606180903-update-endpoint-put-semantics.md) | Accepted | 更新エンドポイント (PUT) はドメイン上ミュータブルな属性のみ置換する |
| [202606190900](202606190900-cross-domain-snapshot.md) | Accepted | 横断データは注文確定時に order 側へスナップショット保存する |
| [202606190901](202606190901-client-generated-code-layout.md) | Accepted | orval 生成物を gen/ ディレクトリに集約する |
| [202606190902](202606190902-parallel-integration-test-template-db.md) | Accepted | 結合テストはテンプレート DB クローンで分離し並列実行する |
| [202606190903](202606190903-repository-cqrs-query-command.md) | Accepted | repository を CQRS で Query / Command に分割する |
| [202606210900](202606210900-what-comment-lint-via-claude-hook.md) | Accepted | what コメント検出を Claude Code hook + LLM 判定で行う |
| [202606211000](202606211000-adr-timestamp-naming.md) | Accepted | ADR の識別子を連番からタイムスタンプ (YYYYMMDDHHmm) に変える |
| [202606211100](202606211100-member-auth-httponly-cookie-session.md) | Accepted | 認証は HttpOnly Cookie + member 所有のサーバ側セッション |
| [202606211200](202606211200-event-driven-shipment-on-payment-settled.md) | Accepted | 決済確定イベントを起点に shipping が配送を手配する (Redis Streams) |
| [202606211520](202606211520-test-case-class-4xx-quasi-normal.md) | Accepted | テストのケース分類で 4xx を準正常系・5xx を異常系に分ける |
| [202606220300](202606220300-frontend-fsd-component-layering.md) | Accepted | フロントエンドのコンポーネントを Feature-Sliced Design で層分けする |
| [202606220716](202606220716-ci-split-by-architecture-quantum.md) | Accepted | CI をアーキテクチャ量子 (顧客系 / backoffice) ごとに 2 ワークフローへ分割する |
| [202606221300](202606221300-merge-mypage-into-store.md) | Accepted | 社外フロントを store に統合し mypage を退役する |
| [202606230930](202606230930-bff-auth-context-and-trust-boundary.md) | Accepted | store BFF に認証コンテキストの集約と X-Member-Id の付与点を置く |
| [202606231000](202606231000-enforce-schema-ownership-with-db-roles.md) | Accepted | データ所有権を schema ごとの最小権限 DB ロールで強制する |
| [202606240522](202606240522-step3-split-db-per-domain-from-weak-edge.md) | Accepted | Step 3 で結合の弱い縁からドメインごとに DB インスタンスを分割する |
