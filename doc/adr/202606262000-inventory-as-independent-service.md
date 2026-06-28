# ADR-202606262000: 在庫を独立サービス (独自 DB・量子) として切り出す

- Status: Accepted
- Date: 2026-06-26
- Supersedes: ADR-[[202606261700]]
- Relates to: ADR-[[202606170900]] (サービス境界), ADR-[[202606240522]] (DB-per-domain), ADR-[[202606231000]] (schema 所有権), ADR-[[202606190900]] (product=order の参照データ), ADR-[[202606211200]] (payment.settled イベント), ADR-[[202606261214]] (DB 制約で冪等化), ADR-[[202606261216]] (致命/縮退分類), GitHub #87

## Context

- ADR-[[202606261700]] は在庫を「既存サービス内の別 schema」に置く前提だったが、このリポジトリには **1 サービスが 2 schema を持つ前例が無い** (1 ドメイン = 1 サービス = 1 DB = 1 量子)。在庫を order/product のどちらに同居させても、その崩れた形か、product=read-only 参照データ (ADR-[[202606190900]]) への checkout 書き込みのどちらかを生む。
- 在庫は入庫・予約・確定・解放という固有のライフサイクルと「売り越さない」固有の不変条件を持つ独立ドメインで、shipping と同じ「運用系・イベント駆動」の形に素直に収まる。

## Decision

在庫を独立サービス (独自 DB `ec_inventory` / ロール `inventory_svc` / internal-private 量子) として切り出す。配置以外の 2 フェーズ・原子的拒否は ADR-[[202606261700]] を引き継ぐが、**台帳の append-only は緩め、予約のライフサイクルは行内の書き込み一度きりタイムスタンプで表す**。

- **入庫は `stock_ins` に追記、予約は `reservations` の 1 行で表す**。確定/解放/期限切れは予約行の nullable タイムスタンプ `confirmed_at` / `released_at` / `expired_at` で持ち、状態はどれが non-null かで導出する。判別列 (kind/status) を持たず、終端の相互排他は `CHECK (num_nonnulls(...) <= 1)` で保証する。
- **導出できる値は行に保存しない**。在庫数カラムを持たず利用可能在庫は集計で導き、予約の期限も `expires_at` 列を持たず `created_at + 固定 TTL` (DB 関数 `inventory.reservation_ttl()`) から導く。
- **引当は予約→確定の2フェーズ**。checkout が `POST /reservations` で TTL 付き予約 → 決済確定 (payment.settled を購読。ADR-[[202606211200]]) で worker が確定へ昇格 → キャンセル/期限切れで解放。
- **超過販売は DB で原子的に防ぐ**。予約時に `pg_advisory_xact_lock(product_id)` で同一商品の同時予約を直列化し、台帳集計で利用可能在庫を算出してから予約行を追記する。在庫不足は致命 (ADR-[[202606261216]]) で checkout を 409 にする。
- **checkout はサービス境界越しに予約する** (order→inventory、edge-proxy 経由)。在庫不足時に注文・決済を残さないため、order は予約失敗で注文を補償取消し、決済手配失敗で予約を解放する (saga)。
- **確定/解放/期限切れは終端 `*_at` が未設定の行だけを更新する冪等な UPDATE**。payment.settled の再配信や二重適用は 0 行更新で吸収する (ADR-[[202606261214]] と同型。check-then-act を WHERE 条件で DB に委ねる)。

## Consequences

- compose / migrate / grant / CI / edge-proxy に inventory を 1 ドメイン分足す (運用対象が増える)。
- order→inventory が同期 HTTP の新依存になる。予約は POST で非冪等のためリトライしない (ADR-[[202606261210]])。在庫不足以外の上流不調は checkout を 502 にする。
- 予約が order_id を相関キーに持つため、checkout は注文を作って order_id を得てから予約する。在庫不足時はその注文を補償で取り消し、「注文・決済を作らない」を結果整合で満たす。
- 予約 TTL の期限切れ回収 (遅延処理) が要る。worker が定期的に期限切れ予約の `expired_at` を立てる。
- 厳密な append-only は緩む (予約行を更新する) が、各 `*_at` は NULL から一度だけ書かれ全タイムラインが残るため履歴は失わない。種別ごとのテーブルや status 列より読み書きが局所で済む。

## Alternatives considered

- **order に在庫 schema を同居** (ADR-[[202606261700]] の一案): checkout と同一 TX で予約でき原子的だが、在庫の真実 (入庫含む) を order が所有する意味的なねじれと、1 サービス 2 schema の前例のない形を持ち込む。
- **product に在庫 schema を同居**: read-mostly カタログへ checkout のホットパス書き込みを生み、ADR-[[202606190900]] の product=参照データ前提とぶつかる (ADR-[[202606261700]] で既に却下)。
- **予約キーを checkout 発番の相関 ID にして注文より先に予約**: 「注文を作らない」を補償なしで満たせるが、payment.settled に相関 ID を載せる payment 側の改修まで波及する。order_id 相関 + 補償に留める。
- **`kind` 判別列の単一台帳 / 終端ごとの別テーブル**: 前者は nullable 列と enum を生む。後者は終端が同型 (`reservation_id` + 時刻) で判別子をテーブル名へ移すだけのうえ、予約の結末取得が多テーブル走査になる。
