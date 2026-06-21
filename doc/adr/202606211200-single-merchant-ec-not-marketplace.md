# ADR-202606211200: 単一事業者の EC（storefront）であり、マーケットプレイスではない

- Status: Accepted
- Date: 2026-06-21

## Context

「単一店舗の EC なのか、複数出店者のマーケットプレイス（モール）なのか」がどこにも明記されて
おらず、UI アプリ `store` の名前を `market` にすべきか、という疑問から前提の不在が判明した。
前提が暗黙だと、名前付け（store / market）やドメイン拡張の判断が毎回「推測」になる。

現状の実装（スキーマ）を計測すると、「売り手（出店者）」を表すデータが**どこにも存在しない**：

- `product.products`: `id, sku, name, price_cents` のみ。`seller_id` / `store_id` が無く、商品は
  単一カタログに属する（誰の出品でもない）。
- `member.members`: `id, email, display_name, password_hash` のみ。buyer/seller の role 区別も
  店舗所有も無い（会員＝買い手の一種別）。ADR-[[202606211100]] の「単一会員基盤」とも整合。
- `"order".orders`: 買い手 `member_id` のみ。`seller_id` が無く、注文は出品者をまたがない。
- `"order".order_items` / `payment.payments` / `shipping.shipments`: 出品者参照・売上分配・
  出荷元店舗の概念がいずれも無い。

マーケットプレイスなら最低限「商品への出品者帰属」「会員の buyer/seller role」「1 カート＝複数
セラーの注文分割」が要るが、3 つとも存在しない。

## Decision

本プロジェクトは **単一事業者の EC（storefront）** を前提とする。**マーケットプレイス（複数
出店者）ではない**。商品カタログ・会員・注文はすべて「1 つの店」に属する。

この帰結として、顧客向け UI アプリの名前は **`store`** とする（`market` / `marketplace` は
複数出店者を含意し実装と乖離するため採らない）。

## Consequences

- `store` という名前は実装（売り手概念の不在）と一致する。命名の根拠がこの ADR に固定される。
- product は単一カタログとして扱ってよい。商品に出品者を紐づける設計はしない。
- 会員は買い手のみ。権限ロール（運営スタッフ等）が必要になっても、それは「出店者」ではなく
  別軸の認可として扱う。

## Alternatives considered

- **マーケットプレイス化（seller ドメイン追加）**: 将来複数事業者を載せる構想が出たら、
  `seller`、商品の出品者帰属、注文の出品者ごと分割などを伴う**ドメイン拡張**として、その時点で
  別 ADR を起こす。名前 `market` への変更はその判断とセットでのみ意味を持つ。
