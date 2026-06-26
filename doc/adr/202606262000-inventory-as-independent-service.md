# ADR-202606262000: 在庫を独立サービス (独自 DB・量子) として切り出す

- Status: Accepted
- Date: 2026-06-26
- Supersedes: ADR-[[202606261700]]
- Relates to: ADR-[[202606170900]] (サービス境界), ADR-[[202606240522]] (DB-per-domain), ADR-[[202606231000]] (schema 所有権), ADR-[[202606190900]] (product=order の参照データ), ADR-[[202606211200]] (payment.settled イベント), ADR-[[202606261214]] (DB 制約で冪等化), ADR-[[202606261216]] (致命/縮退分類), GitHub #87

## Context

- ADR-[[202606261700]] は在庫を「既存サービス内の別 schema」に置く前提だったが、このリポジトリには **1 サービスが 2 schema を持つ前例が無い** (1 ドメイン = 1 サービス = 1 DB = 1 量子)。在庫を order/product のどちらに同居させても、その崩れた形か、product=read-only 参照データ (ADR-[[202606190900]]) への checkout 書き込みのどちらかを生む。
- 在庫は入庫・予約・確定・解放という固有のライフサイクルと「売り越さない」固有の不変条件を持つ独立ドメインで、shipping と同じ「運用系・イベント駆動」の形に素直に収まる。

## Decision

在庫を独立サービス (独自 DB `ec_inventory` / ロール `inventory_svc` / internal-private 量子) として切り出す。配置以外の在庫設計 (台帳・2 フェーズ・原子的拒否) は ADR-[[202606261700]] の判断を引き継ぐ。

- **append-only の在庫変動台帳を単一情報源にする**。入庫 (`stock_ins`) / 予約 (`reservations`) / 確定 (`confirmations`) / 解放 (`releases`) を**種別ごとのテーブルに追記**し (判別列 `kind` を持たない)、在庫数カラムも持たず利用可能在庫は集計で導く。
- **引当は予約→確定の2フェーズ**。checkout が `POST /reservations` で TTL 付き予約 → 決済確定 (payment.settled を購読。ADR-[[202606211200]]) で worker が確定へ昇格 → キャンセル/期限切れで解放。
- **超過販売は DB で原子的に防ぐ**。予約時に `pg_advisory_xact_lock(product_id)` で同一商品の同時予約を直列化し、台帳集計で利用可能在庫を算出してから予約行を追記する。在庫不足は致命 (ADR-[[202606261216]]) で checkout を 409 にする。
- **checkout はサービス境界越しに予約する** (order→inventory、edge-proxy 経由)。在庫不足時に注文・決済を残さないため、order は予約失敗で注文を補償取消し、決済手配失敗で予約を解放する (saga)。
- **確定/解放の冪等性は DB 制約に委ねる** (ADR-[[202606261214]] と同型)。`confirmations` / `releases` の `reservation_id` 主キーで予約 1 件につき高々 1 行に制限し、payment.settled の再配信を `ON CONFLICT DO NOTHING` で吸収する。

## Consequences

- 在庫が独立量子になり、shipping と対称な「運用系 HTTP + worker (payment.settled 購読)」構成に揃う。compose / migrate / grant / CI / edge-proxy に inventory を 1 ドメイン分足す。
- order→inventory が同期 HTTP の新依存になる。予約は POST で非冪等のためリトライしない (ADR-[[202606261210]])。在庫不足以外の上流不調は checkout を 502 にする。
- 予約が order_id を相関キーに持つため、checkout は注文を作って order_id を得てから予約する。在庫不足時はその注文を補償で取り消し、「注文・決済を作らない」を結果整合で満たす。
- 予約 TTL の期限切れ回収 (遅延処理) が要る。worker が定期的に期限切れ予約へ解放を追記する。

## Alternatives considered

- **order に在庫 schema を同居** (ADR-[[202606261700]] の一案): checkout と同一 TX で予約でき原子的だが、在庫の真実 (入庫含む) を order が所有する意味的なねじれと、1 サービス 2 schema の前例のない形を持ち込む。
- **product に在庫 schema を同居**: read-mostly カタログへ checkout のホットパス書き込みを生み、ADR-[[202606190900]] の product=参照データ前提とぶつかる (ADR-[[202606261700]] で既に却下)。
- **予約キーを checkout 発番の相関 ID にして注文より先に予約**: 「注文を作らない」を補償なしで満たせるが、payment.settled に相関 ID を載せる payment 側の改修まで波及する。order_id 相関 + 補償に留める。
