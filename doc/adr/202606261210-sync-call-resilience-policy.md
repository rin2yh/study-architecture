# ADR-202606261210: order の同期呼び出しに timeout・リトライ・サーキットブレーカを入れる

- Status: Accepted
- Date: 2026-06-26
- Relates to: ADR-[[202606241356]] (可観測性で効果を計測), ADR-[[202606261214]] (冪等性。POST リトライ解禁の前提), ADR-[[202606211520]] (テスト分類)

## Context

- order は checkout で product / payment を同期 HTTP で呼ぶ (`server/order/internal/gateway/gateway.go`)。timeout 以上の保護が無く、下流の一時障害が `ErrUpstream` で checkout 全体の失敗に直結する。
- Step 4 で計測基盤が入り (ADR-[[202606241356]])、リトライ回数・ブレーカ開閉・改善前後を観測できるようになった。耐性を効果を見ながら入れられる。

## Decision

order の出力ポート (gateway) に timeout / リトライ / サーキットブレーカ / フォールバックを入れる。

- **サーキットブレーカは sony/gobreaker**。Go の CB のデファクトで軽量・generics・MIT、`otelhttp` 計装に干渉しない。
- **リトライ (指数バックオフ + ジッタ) は `server/internal/httpx` に自前の薄い実装**。各耐性パターンの挙動を読めるようにし依存を増やさない (CLAUDE.md「推測するな、計測せよ」)。
- **リトライ対象は冪等な呼び出しに限る**。GET (order→product) はリトライ可。非冪等な POST (order→payment 決済作成) は素朴なリトライが二重決済を生むため、**冪等性 (ADR-[[202606261214]]) が入るまでリトライしない**。timeout と CB は POST にも効かせる。
- CB は呼び出し先 (product / payment) ごとに分け、開いている間はフォールバック (即時エラー) を返す。

## Consequences

- 下流の一時障害でリトライ・CB が効き、checkout が即失敗しなくなる。効果は Grafana で観測できる。
- POST のリトライ保留により、payment 一時障害時の自動回復は冪等性 (5-3) 完了まで限定的。
- gobreaker への依存が 1 つ増える。リトライは自前のため状態遷移・並行のテストを持つ。

## Alternatives considered

- **failsafe-go で一括 (Retry/CB/Timeout/Fallback の合成)**: 手書きは減るが若いライブラリ依存と DSL が増え、現要件には過剰。将来 bulkhead/hedge まで広げるなら再検討。
- **自前で全部**: 依存ゼロだが CB の状態遷移・並行性を自前保守する負担に見合わない。CB はデファクトに乗る。
- **POST も今すぐリトライ**: 冪等性キー無しでは二重決済リスク。冪等にしてからが順序。
