# 学習ロードマップ — マイクロサービス化とアーキテクチャ特性の深化

現状（Step 0〜3 完了後）の到達点を棚卸しし、マイクロサービスへ進化させる場合に不足している
要素を「追加実装できる単位」で列挙する。着手時は従来どおり ADR を起こし、1 PR 1 テーマで進める。

## 現在地（すでに獲得済みの特性）

- サービス分割: ドメイン 6 サービス（product / order / payment / member / shipping / inventory）、量子ごとの CI 分割（ADR-[[202606220716]]）
- データ所有権: ドメインごとの DB インスタンス分割完了（ADR-[[202606240522]]）、schema ロールによる強制（ADR-[[202606231000]]）、横断はスナップショット（ADR-[[202606190900]]）
- 非同期連携: Redis Streams + Transactional Outbox（専用テーブル + 共有ディスパッチャ、ADR-[[202606300600]]）、イベント駆動の補償（ADR-[[202606261702]]）、冪等性（ADR-[[202606261214]]）
- 同期呼び出しの耐障害性: timeout / リトライ / サーキットブレーカ（ADR-[[202606261210]]）、致命 / 縮退可の分類（ADR-[[202606261216]]）
- o11y: OTel + Alloy + Tempo / Loki / Prometheus / Grafana（ADR-[[202606241356]]）、trace-log 相互参照、span link（ADR-[[202606250159]]）、マスキング（ADR-[[202606250141]]）、RED ダッシュボード / アラート（ADR-[[202606261100]]）、cAdvisor（ADR-[[202606261600]]）
- 信頼境界: BFF による認証集約と X-Member-Id 付与（ADR-[[202606230930]]）、edge-proxy、顧客系 / 運用系のネットワーク分離（ADR-[[202606170909]]）

サービスベースアーキテクチャとしてはほぼ完成形で、マイクロサービスとの残差は
「独立デプロイの安全網」「スケールアウト」「運用の自動化・検証」に集約される。

## Track A: メッセージングの信頼性（不足機能・優先度高）

### A-1. Poison message 対応と DLQ（実装候補の筆頭）

現状の consumer（payment / shipping / inventory）は処理失敗時に ACK せず PEL に残すが、
`XReadGroup ">"` は新規メッセージしか配らないため、**失敗したメッセージは二度と再処理されない**
（同名 consumer の再起動でも `>` 読みのため引き取られない）。壊れた payload が来ると黙って滞留する。

- `XAUTOCLAIM` による min-idle 経過分の引き取りループを共有実装（`server/internal/redisx`）に追加
- 配送回数（`XPENDING` の delivery count）上限を設け、超過分は DLQ ストリームへ退避 + アラート
- DLQ の中身を backoffice から確認・再投入できると運用学習としてさらに良い

### A-2. イベントスキーマの進化（versioning）

イベント payload に version がなく、producer と consumer を独立デプロイすると互換性が壊れうる。
イベントへの version フィールド導入、後方互換の変更ルール（フィールド追加のみ可など）を ADR 化し、
互換性テストで CI 担保する。マイクロサービスの「独立デプロイ」の核心。

### A-3. Saga オーケストレーション（比較学習）

補償は choreography（ADR-[[202606261702]]）で実装済み。checkout の一連（在庫予約 → 決済 →
確定）を orchestrator 型で書き直すか、別ユースケース（返品など）を orchestrator で足し、
両スタイルの利害（可視性 vs 結合）を体験比較する。

## Track B: スケールアウト（分散システムの本丸）

### B-1. 複数レプリカ運用

現在は全サービス 1 インスタンス。`deploy.replicas: 2` にするだけで学習課題が一気に顕在化する:

- outbox ディスパッチャの競合（`FOR UPDATE SKIP LOCKED` の有無・二重送出）
- consumer 名が hostname 依存であることの影響（PEL の孤児化と A-1 の必然性）
- edge-proxy / BFF からの負荷分散、コネクションプールのサイズ設計

「計測して壊れる箇所を見つけ、直す」演習として最も費用対効果が高い。

### B-2. edge の保護（rate limiting / load shedding）

nginx に `limit_req` が無く、過負荷時は無制限に後段へ流れる。edge での rate limit、
アプリ側の同時実行上限（bulkhead）、負荷時に縮退可機能から落とす load shedding を段階導入する。

## Track C: 独立デプロイの安全網

### C-1. 契約テスト（Consumer-Driven Contract）

OpenAPI からの codegen で「同一リポジトリ内の同期」は取れているが、**デプロイ単位の独立性**を
守る契約テストがない。BFF↔各サービス、order↔payment / inventory の消費側期待を契約として
固定し、provider 側 CI で検証する（Pact、または openapi diff による破壊的変更検出から始めても良い）。

### C-2. API バージョニング / 非推奨化の運用ルール

破壊的変更をどう告知・移行するか（sunset ヘッダ、v2 並走、イベントは A-2）を ADR 化する。

### C-3. Kubernetes への移行（kind / k3d でローカル完結）

compose → k8s は運用面の学習が最も濃い区間。service discovery、readiness と liveness の
使い分け（現状 `/healthz` は liveness のみで、依存 DB 断を反映する readiness が無い）、
rolling update と graceful shutdown の検証、HPA。費用ゼロ方針は kind で維持できる。

### C-4. Progressive delivery

canary / feature flag。k8s 後に Argo Rollouts か、手前なら BFF でのフラグ分岐から。

## Track D: o11y とアーキテクチャ特性の深化

### D-1. SLO / SLI とバーンレートアラート

現状は RED の閾値アラートのみ。サービスごとに SLO（例: checkout 可用性 99.5% / 30 日）を定義し、
エラーバジェットと multi-window burn rate アラートへ置き換える。アラート疲れと過検知の学習に直結。

### D-2. 負荷試験と計測（「推測するな、計測せよ」の実践）

k6 で checkout フローの負荷シナリオを常備し、B-1 / B-2 の変更前後で p99 / スループット /
飽和点を計測して比較する。ボトルネック特定 → 改善 → 再計測のループを回す土台。

### D-3. カオス実験（resilience の検証）

CB・リトライ・縮退（Track 済み実装）が**実際に働くか**を、toxiproxy 等で遅延・断を注入して
計測で確かめる。「実装した」と「機能する」を区別する演習。

### D-4. テレメトリの高度化

- Exemplars: メトリクスのパネルから該当トレースへ直接ジャンプ
- Tail-based sampling: エラー / 遅延トレースを優先保存（トラフィック増加後に効く）
- Continuous profiling（Pyroscope）: CPU / メモリのボトルネックをトレースと突き合わせ
- Synthetic monitoring: blackbox exporter で外形監視

## Track E: セキュリティ・ガバナンス

### E-1. サービス間認証（zero trust 化）

X-Member-Id はネットワーク分離への信頼に依存する（ADR-[[202606230930]] が明示するトレードオフ）。
BFF 発行の短命 JWT を内部伝播する、または mTLS（k8s 後なら service mesh）で「ネットワークに
いる = 信頼」を外す。

### E-2. アーキテクチャ適応度関数（fitness function）

依存境界（サービス間 import 禁止、`internal/` 越境、handler→db 直接参照禁止など）を
depguard / go-arch-lint で CI 検査し、ADR の決定を機械的に守る。『進化的アーキテクチャ』の実践。

### E-3. Secrets 管理

DSN / 資格情報の env 直書きからの脱却（compose secrets → k8s Secret / 外部化）。

## 推奨着手順

| 順 | テーマ | 理由 |
| --- | --- | --- |
| 1 | A-1 DLQ / 再配送 | 現状の実挙動の穴。小さく閉じて効果が明確 |
| 2 | D-2 負荷試験基盤 (k6) | 以降の全変更を「計測で」評価する土台になる |
| 3 | B-1 複数レプリカ化 | 分散の課題が一気に顕在化し、A-1 の検証にもなる |
| 4 | D-1 SLO / バーンレート | アラートの質を上げ、以降の実験の判定基準になる |
| 5 | C-1 契約テスト | 独立デプロイ（マイクロサービスの定義的性質）の安全網 |
| 6 | C-3 k8s 移行 | 運用系学習の本丸。1〜5 の資産をそのまま持ち込める |

以降は A-2 / A-3 / B-2 / C-4 / D-3 / E 系を関心に応じて。いずれも着手時に ADR を起こすこと。
