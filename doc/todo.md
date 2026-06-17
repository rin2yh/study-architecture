# 残タスク

最終更新: 2026-06-17

## 既知の不具合

- [ ] **Docker での UI SSR 失敗**（最優先）
      TanStack Start (Nitro 3 beta) の SSR self-request が Docker(OrbStack) の network と
      非互換でハング/500。詳細は [known-issues.md](known-issues.md)。
      候補: Nitro preset/server オプション調整 / TanStack Start 安定版(1.0)待ち /
      SSR をやめて CSR + TanStack Query に切替（ADR 0006 見直し）。
      ※ backend 5サービス + migrate の Docker、および host の dev SSR は動作確認済み。

## コード品質

- [ ] mutator の `res.json()` 由来の暗黙 `any` を `unknown` 化（明示 any は 0 件、
      oxlint `no-explicit-any` は適用済み）。
- [ ] UI のエラー/ローディング表示（ローダ失敗時のフォールバック）。
- [ ] backend のバリデーション・エラーレスポンス整備（現状は最小）。

## テスト / CI

- [ ] GitHub Actions で `mise ci`（fmt:check / vet / test）+ `mise cover` + client の
      lint/typecheck/build を回す。
- [ ] カバレッジは現状 server 60.5% / client(@ec/api 100%・app 62.5%)。閾値を CI で担保。
- [ ] backend の repository/handler を実 DB 統合テスト（testcontainers 等）で補強。

## 機能の肉付け（Step 0 の範囲内）

- [ ] 各サービスの一覧以外（取得/作成/更新）エンドポイントと業務ロジック。
- [ ] order のカート/チェックアウト導線、payment・shipping との連携。
- [ ] 会員認証（member）と UI のログイン状態。

## ロードマップ（compose の進化で進める）

- [ ] **Step 1**: API ファサード（UI の BFF）を足し UI 入口を集約。段取りは各サービスに残す。
- [ ] **Step 2**: データ所有権を確定し schema 分離を徹底（他ドメインへの書き込みは持ち主経由）。
- [ ] **Step 3**: 結合の弱い縁から DB 分割（payment / shipping → member → product × order）。
      横断データ（注文時の商品価格・名称）は (a) サービス間参照 か (b) スナップショット保存。

## 設計判断（必要時に ADR 追記）

- [ ] DB 分割時の横断データ戦略（ADR）。
- [ ] 認証方式（セッション / JWT）と member サービスの責務（ADR）。
