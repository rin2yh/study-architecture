# ADR-202606180903: 更新エンドポイント (PUT) はドメイン上ミュータブルな属性のみ置換する

- Status: Accepted
- Date: 2026-06-18
- Related: ADR-[[202606180901]] (API エラーモデル / 取得・作成), ADR-[[202606170901]] (codegen-first), ADR-[[202606170903]] (schema 分離)

## Context

ADR-[[202606180901]] で全 5 サービスに取得 (`GET /<resource>/{id}`) と作成 (`POST /<resource>`) を
足し、「更新 (PUT) は作成と同じ束縛/検証パターンで後から各サービスに足せる」と書いた
(issue #4)。実際に更新を足すにあたり、2 つの判断が要る:

1. **PUT で何を更新可能にするか** — 「作成と同じ全項目」を機械的に置換可能にすると、
   ドメイン上*書き換えてはいけない値*まで可変になる。具体的には:
   - **所有関係を表す外部キー** (`order.memberId` / `payment.orderId` / `shipping.orderId`)
     — 付け替えると参照が壊れる。
   - **導出値・取引の事実** (`order.totalCents` は明細からの導出、`payment.amountCents` /
     `payment.method` は発生済み取引の記録) — 後から編集すると会計が破綻する。訂正は
     返金・再決済など別操作で表現すべき。
   - **安定識別子** (`product.sku`) — 注文・在庫から参照される外部キー的な値で、変えると
     参照不整合になる。
2. **更新失敗のエラー正規化** — `UPDATE ... RETURNING` は対象行が無いと no rows になり
   (404 相当)、unique 列を更新する場合は unique_violation (409 相当) にもなる。取得用の
   `FromRead` (no rows → NotFound) でも作成用の `FromWrite` (23505 → Conflict) でも片方しか
   拾えない。

## Decision

### 1. 更新可能なのは「ドメイン上ミュータブルな属性」だけ

更新リクエスト (`Update<Resource>Request`) は、id・生成時刻・所有 FK・導出値・取引の事実・
安定識別子を**すべて除いた、ドメイン上変えてよい属性**だけを持つ。PUT はその可変サブセットを
全置換する (部分更新の PATCH ではない → §3)。

| サービス | 更新可能 (PUT body)   | 不変 (理由)                                                      |
| -------- | --------------------- | --------------------------------------------------------------- |
| product  | name, priceCents      | id, createdAt, **sku (安定識別子)**                             |
| order    | status                | id, createdAt, memberId (FK), **totalCents (導出値)**           |
| payment  | status                | id, createdAt, orderId (FK), **amountCents / method (取引の事実)** |
| member   | email, displayName    | id, createdAt                                                   |
| shipping | status                | id, createdAt, orderId (FK), **carrier / trackingNo (出荷の事実)** |

- `order` / `payment` / `shipping` は実質 **`status` のみ**の更新になる。これは
  「ステータス遷移＝状態機械」というまっとうな更新像で、金額や出荷情報は一度記録したら
  別操作 (返金・再出荷など) で扱う。
- 検証は ADR-[[202606180901]] の規則を踏襲: 構文/形式 → 400 (gin binding タグ)、意味的検証 → 422
  (`middleware.Unprocessable`)、unique 違反 → 409、未存在 → 404。更新で 422 が残るのは
  `product.priceCents` の負値チェックのみ (他は数値項目を更新しないため)。

### 2. 更新専用の正規化 `dberr.FromUpdate` を共通パッケージに足す

`UPDATE ... RETURNING` のエラーを、no rows → `ErrNotFound`、unique_violation → `ErrConflict`
の両方に対応づける `dberr.FromUpdate` を追加し、repository の `Update<Resource>` が一様に使う
(`FromRead` の no-rows 判定 + `FromWrite` の委譲)。実際に 409 が起こりうるのは unique 列を
更新する **`member.email` だけ**だが、正規化を一様にしておくことで「更新が読み取り・書き込み
双方で失敗しうる」性質を 1 箇所に閉じ込める。handler 側は到達しうるエラーだけ分岐する
(member の更新のみ `ErrConflict` を 409 に対応づけ、他は `ErrNotFound` のみ)。

### 3. 動詞は PUT (全置換) のまま

可変サブセットを全項目必須で置換する冪等な操作なので動詞は PUT。`status` のみのリソースは
1 項目の表現を置換する PUT になる。「送られた項目だけ更新する」部分更新 (PATCH) は採らない
(理由は Alternatives)。

## Consequences

- **不変条件が型と契約に出る**: 金額・取引手段・安定識別子・所有 FK が PUT body に無いので、
  API 利用者も生成クライアント (orval) も「注文金額や決済額は更新で書き換えられない」を
  契約として読める。データ所有権を確定していく Step 2/3 (ADR-[[202606170903]]) とも整合する。
- **更新はほぼ状態遷移に収束**: order/payment/shipping が `status` のみになり、更新 API の
  意味が「ライフサイクルを進める」に明確化される。
- **エラー対応が崩れない**: 更新の 404/409 が `FromUpdate` 1 関数に集約。FK・不変列を更新
  しないので外部キー違反 (23503) は通常起きず、起きても透過されて 500 (隠すべき内部エラー)。
- **将来の余地**: 状態遷移の制約 (例: `delivered` から `pending` へ戻せない) など業務ルール
  に踏み込んだ検証は handler に 422 として足せる。今回は Step 0 の素直な置換に留める。

## Alternatives considered

- **作成と同項目を全置換 (FK・金額含む)**: 実装と生成は最も単純で対称的だが、所有関係の
  付け替えや金額・取引手段の事後編集を許してしまい、会計・参照の整合がドメイン上壊れる。
  学習用途でも誤った API 像を与えるため見送り。
- **金額や出荷情報も更新可のまま (FK だけ不変)**: 当初案。`totalCents` / `amountCents` /
  `method` / `sku` が書き換え可能で、導出値・取引の事実・安定識別子を可変にしてしまうため
  却下。本 ADR で「ドメイン上ミュータブルな属性のみ」に絞り直した。
- **PATCH で部分更新**: 「送られた項目だけ更新」は nullable/optional とゼロ値の区別
  (sqlc の `COALESCE` やポインタ束縛) が要り、Step 0 の薄い骨格には重い。全置換 PUT に倒す。
- **`FromUpdate` を作らず handler 側で no rows と 23505 を個別判定**: db 依存 (pgx) が
  handler に漏れる。ADR-[[202606180901]] で `dberr` に正規化を寄せた方針に反するため、更新用も
  `dberr` に足す。
