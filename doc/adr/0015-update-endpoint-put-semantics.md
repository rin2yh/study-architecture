# ADR 0015: 更新エンドポイント (PUT) は業務項目のみ置換し FK は不変とする

- Status: Proposed
- Date: 2026-06-18
- Related: [[0014]] (API エラーモデル / 取得・作成), [[0002]] (codegen-first), [[0004]] (schema 分離)

## Context

[[0014]] で全 5 サービスに取得 (`GET /<resource>/{id}`) と作成 (`POST /<resource>`) を
足し、「更新 (PUT) は作成と同じ束縛/検証パターンで後から各サービスに足せる」と書いた
(issue #4)。実際に更新を足すにあたり、2 つの判断が要る:

1. **PUT で何を更新可能にするか** — 作成リクエストと同じ全項目を置換するか、一部を不変
   とするか。`order.memberId` / `payment.orderId` / `shipping.orderId` のような所有関係を
   表す外部キーを後から付け替えられると、ドメイン的に不自然になる。
2. **更新失敗のエラー正規化** — `UPDATE ... RETURNING` は対象行が無いと no rows になり
   (404 相当)、同時に unique 列の付け替えで unique_violation (409 相当) にもなる。取得用の
   `FromRead` (no rows → NotFound) でも作成用の `FromWrite` (23505 → Conflict) でも片方しか
   拾えない。

## Decision

### 1. PUT は業務項目のみ全置換し、FK と id / createdAt は不変

更新リクエスト (`Update<Resource>Request`) は、作成リクエストから **id・生成時刻・所有
関係を表す外部キーを除いた業務項目** だけを持つ。PUT は対象リソースの業務項目を全置換する
(部分更新の PATCH ではない)。

| サービス | 更新可能 (PUT body)            | 不変 (path id / 生成値 / FK)     |
| -------- | ----------------------------- | -------------------------------- |
| product  | sku, name, priceCents         | id, createdAt                    |
| order    | status, totalCents            | id, createdAt, **memberId**      |
| payment  | amountCents, method, status   | id, createdAt, **orderId**       |
| member   | email, displayName            | id, createdAt                    |
| shipping | carrier, trackingNo, status   | id, createdAt, **orderId**       |

- 検証は [[0014]] の規則をそのまま使う: 構文/形式 → 400 (gin binding タグ)、意味的検証
  (金額の負値) → 422 (`middleware.Unprocessable`)、unique 違反 → 409、未存在 → 404。
- product / member は所有 FK を持たないため、更新可能項目は作成と同じになる。これは
  「FK を除く」原則の結果であって、作成と同一にすること自体が目的ではない。

### 2. 更新専用の正規化 `dberr.FromUpdate` を共通パッケージに足す

`UPDATE ... RETURNING` のエラーを、no rows → `ErrNotFound`、unique_violation → `ErrConflict`
の両方に対応づける `dberr.FromUpdate` を追加する。中身は `FromRead` の no-rows 判定 +
`FromWrite` の委譲で、更新が読み取り (NotFound) と書き込み (Conflict) 双方で失敗しうる
という性質を 1 箇所に閉じ込める。repository の `Update<Resource>` はこれを使う。

## Consequences

- **不変条件が型に出る**: 所有 FK が PUT body に無いので、API 利用者も生成クライアント
  (orval) も「注文の持ち主は付け替えられない」を契約として読める。データ所有権を確定して
  いく Step 2/3 ([[0004]]) とも整合する。
- **エラー対応が崩れない**: 更新の 404/409 が `FromUpdate` 1 関数に集約され、handler は
  `ErrNotFound` / `ErrConflict` を見るだけ。FK 不変なので外部キー制約違反 (23503) は通常
  起きず、起きても透過されて 500 になる (隠すべき内部エラー扱い)。
- **将来の余地**: 状態遷移の制約 (例: `delivered` から `pending` へ戻せない) など、業務
  ルールに踏み込んだ検証は handler に 422 として足せる。今回は Step 0 の素直な置換に留める。

## Alternatives considered

- **作成と同項目を全置換 (FK 含む)**: 実装と生成は最も単純で対称的だが、所有関係を後から
  付け替えられてしまい意味的に不自然。学習用途でも誤った API 像を与えるため見送り。
- **PATCH で部分更新**: 「送られた項目だけ更新」は nullable/optional とゼロ値の区別
  (sqlc の `COALESCE` やポインタ束縛) が要り、Step 0 の薄い骨格には重い。全置換 PUT に倒す。
- **`FromUpdate` を作らず handler 側で no rows と 23505 を個別判定**: db 依存 (pgx) が
  handler に漏れる。[[0014]] で `dberr` に正規化を寄せた方針に反するため、更新用も
  `dberr` に足す。
