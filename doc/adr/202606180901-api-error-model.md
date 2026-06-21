# ADR-202606180901: API エラーモデルを共通 Error スキーマ + ErrorJSON ミドルウェアに集約する

- Status: Accepted
- Date: 2026-06-18
- Related: ADR-[[202606170901]] (codegen-first), ADR-[[202606170907]] (Gin)

## Context

Step 0 時点では、各サービスのエラー整形は `server/internal/middleware/ErrorJSON()`
で 400 (`gin.ErrorTypeBind|Public` で文言透過) / 500 (内部詳細を隠す) の 2 値だけを
扱っていた。一方で OpenAPI 仕様にはエラーレスポンスの定義が無く、

- クライアント (orval) はエラー時の body 型を知らない
- 404 / 409 / 422 のようなドメインのセマンティクスを表現する手段が無い
- エラー JSON の shape (`code` / `message`) が仕様化されておらず、サーバ実装の暗黙知

という状態だった (issue #2)。

## Decision

### 1. 共通 Error スキーマと default レスポンス

5 サービスすべての `openapi.yaml` に共通の `Error` スキーマと、再利用可能な
`components.responses.Error` を定義し、各オペレーションに `default` レスポンスとして
紐づける。

```yaml
components:
  responses:
    Error:
      description: エラー (全サービス共通フォーマット)
      content:
        application/json:
          schema: { $ref: "#/components/schemas/Error" }
  schemas:
    Error:
      type: object
      required: [code, message]
      properties:
        code: { type: string }     # 機械可読なエラー種別
        message: { type: string }  # 人間可読なエラー説明
```

- `code` / `message` は `ErrorJSON` が返す JSON と shape を一致させる。これにより
  oapi-codegen は Go の `api.Error` 型を、orval は TS の `Error` / `ErrorResponse` 型を
  生成し、サーバ/クライアント双方で同じ契約を共有する (ADR-[[202606170901]])。

### 2. ステータスのマッピングは ErrorJSON 1 箇所に集約

HTTP ステータスと `code` の対応づけは引き続き共通ミドルウェア `ErrorJSON` に集約する
(ADR-[[202606170907]])。handler は `c.Error(err)` でエラーを積むだけで、整形・隠蔽・ログレベルの
判断はミドルウェアが行う。マッピング規則:

| 入力 (handler が積むエラー)          | status | code                   | message       |
| ------------------------------------ | ------ | ---------------------- | ------------- |
| `*middleware.AppError`               | 表明値 | 表明値                 | 表明値 (透過) |
| `gin.ErrorTypeBind` / `ErrorTypePublic` | 400 | `bad_request`          | 透過          |
| それ以外                             | 500    | `internal`             | 固定文言で隠蔽 |

### 3. 404 / 409 / 422 のセマンティクスを型付きエラーで表現

400 / 500 の 2 値しか出せなかった gap を埋めるため、型付きの `middleware.AppError`
(`Status` / `Code` / `Message`) とコンストラクタを追加する:

- `NotFound(msg)`       → 404 `not_found`
- `Conflict(msg)`       → 409 `conflict`
- `Unprocessable(msg)`  → 422 `unprocessable_entity`
- `NewError(status, code, msg)` → 任意

handler は `c.Error(middleware.NotFound("member 99 not found"))` のように積む。
ログは status で振り分ける (5xx は Error、4xx は Warn)。

### 4. request body バリデーションと書き込みエンドポイント

issue #2 の「request body の zod / oapi-codegen バリデーション」を、全 5 サービスに
取得 (`GET /<resource>/{id}`) と作成 (`POST /<resource>`) のエンドポイントを足して
実装した。これにより 404 / 409 / 422 が実際に使われ、机上の status コードにならない。

- **構文・形式の検証 → 400**: request body のスキーマに `x-oapi-codegen-extra-tags`
  で gin binding タグ (`binding: required` / `required,email` / `required,gt=0`) を載せ、
  oapi-codegen が生成する struct タグへ反映する。handler は `c.ShouldBindJSON` で束縛し、
  失敗を `gin.ErrorTypeBind` として積む → `ErrorJSON` が 400 `bad_request` に整形。
  orval も同じ OpenAPI から request 用 zod スキーマを生成するため、クライアント側でも
  同じ制約で検証できる。
- **意味的検証 → 422**: 構文は妥当だが業務的に不正な入力 (金額が負など) は handler が
  `middleware.Unprocessable(...)` で 422 を積む。`order.totalCents` / `payment.amountCents`
  / `product.priceCents` の負値で使用。
- **一意制約違反 → 409**: postgres の unique_violation (SQLSTATE 23505) を repository が
  共通パッケージ `server/internal/dberr` で `ErrConflict` に正規化し、handler が
  `middleware.Conflict(...)` で 409 に対応づける。`member.email` / `product.sku` で使用。
- **未存在 → 404**: 取得で行が無い (no rows) を `dberr.ErrNotFound` に正規化し、handler が
  `middleware.NotFound(...)` で 404 に対応づける。全サービスの `GET /{id}` で使用。

`dberr` は pgx 固有のエラー (`pgx.ErrNoRows` / `pgconn.PgError`) をセンチネルエラーに正規化する
共通パッケージで、db 依存を handler に漏らさず 5 サービスで重複なく使い回す。更新 (PUT) は
作成と同じ束縛/検証パターンで後から各サービスに足せる。

## Consequences

- **契約の明文化**: エラー body の形 (`code` / `message`) が OpenAPI に乗り、生成型として
  サーバ/クライアントに伝播する。クライアントは `default` レスポンス型でエラーを型安全に
  扱える。
- **ドメインセマンティクスの一元管理**: 404 / 409 / 422 を出すかどうかは handler が
  `AppError` を選ぶだけ。status と code の対応は `ErrorJSON` の 1 箇所だけを見れば分かる。
- **500 の安全性は維持**: `AppError` 以外・非 Public エラーは従来どおり内部詳細を隠して
  500 を返す。`AppError` で明示的に組み立てたものだけが文言を透過する。
- **クライアント生成の副作用**: `default` レスポンスを足すと orval (fetch client) は各 tag
  ファイルに `HTTPStatusCode*` 型を生成する。tags-split で system / リソースの 2 tag が
  同名型を出すため、サービス別 barrel (`src/<svc>.ts`) で `HTTPStatusCode*` を明示
  re-export して star-export の重複 (TS2308) を解消している。

## Alternatives considered

- **handler ごとに status/JSON を直接書く**: ミドルウェア集約をやめると、隠蔽漏れ
  (内部エラー文言の露出) や shape のばらつきが起きやすい。横展開コストも上がる。
- **エラーコードを HTTP status と 1:1 に固定**: 同じ 409 でも「一意制約」「状態遷移不正」
  など複数あり得る。`code` を string にして status とゆるく対応づけることで、後から
  細分化できる余地を残す。
- **`includeHttpResponseReturnType: false` で orval のレスポンス型を素の data に倒す**:
  `HTTPStatusCode*` の生成を避けられるが、既存の `{ data, status, headers }` 契約
  (mutator / UI ローダが依存) を壊すため見送り。barrel の明示 re-export で対処した。
