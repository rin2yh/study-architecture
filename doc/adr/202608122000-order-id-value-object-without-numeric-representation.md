# ADR-202608122000: 注文 ID を値オブジェクトにし、数値表現をデータ層に閉じる

- Status: Accepted
- Date: 2026-08-12
- Relates to: ADR-[[202606261702]]

## Context

- 注文 ID は `int64` の裸値でイベント・DB・HTTP を渡り歩いており、`paymentID` と取り違えても
  コンパイルが通る。文字列からの復元も orderevent と paymentevent に重複していた。
- `int64` である理由は DB の bigint 採番であって、ドメインの都合ではない。

## Decision

- 注文 ID は order が所有し、`server/internal/order` の `ID` (未公開フィールド) で表す。生成は
  `Parse` / `ParseIDFromEvent` だけを通す。
- **ドメインの公開 API に数値表現を出さない**。正準表現は文字列とし、`int64` への変換は生成コード
  (sqlc / OpenAPI クライアント) に接する層だけが行う。
- 同じ理由で `outbox.Event.AggregateID` も文字列にし、bigint 列への変換は各サービスの outbox
  アダプタに置く。

## Consequences

- 取り違えと未検証値の混入がコンパイル時に止まる。既存のイベントテストが `OrderID: 20` と書けなくなり、
  生成口を通る形へ矯正された。
- 変換は `strconvx.MustInt64` (失敗は panic) を使う。渡るのは生成口を通った ID だけなので、error を
  返すと呼び出し側に扱えない分岐が増えるだけになる。
- 変換ヘルパーはサービスごとに重複する。`outboxInserter` と同じくサービス間の独立を優先する。
- HTTP ハンドラ経路 (`Reserve` / `GetOrderItems` など) は裸の `int64` のまま。揃えるかは別途。

## Alternatives considered

- **named type (`type ID int64`)** — アクセサは不要になるが `order.ID(999)` で生成口を迂回でき、
  担保が lint 頼みになる。
- **`Int64()` を残す** — 変換は 1 メソッドで済むが、DB 由来の表現がドメインの公開 API に残る。
- **sqlc の `overrides` で生成コードを `order.ID` にする** — 署名から `int64` は消えるが、
  `driver.Valuer` / `Scanner` の実装、つまり DB の知識がドメイン型に入る。
