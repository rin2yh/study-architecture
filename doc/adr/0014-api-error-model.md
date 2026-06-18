# ADR 0014: API エラーモデルを共通 Error スキーマ + ErrorJSON ミドルウェアに集約する

- Status: Proposed
- Date: 2026-06-18
- Related: [[0002]] (codegen-first), [[0010]] (Gin)

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
  生成し、サーバ/クライアント双方で同じ契約を共有する ([[0002]])。

### 2. ステータスのマッピングは ErrorJSON 1 箇所に集約

HTTP ステータスと `code` の対応づけは引き続き共通ミドルウェア `ErrorJSON` に集約する
([[0010]])。handler は `c.Error(err)` でエラーを積むだけで、整形・隠蔽・ログレベルの
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

### 4. スコープ外 (将来タスク)

issue #2 の「request body の zod / oapi-codegen バリデーション」は、取得/作成/更新の
**write 系エンドポイントを足すタイミング** で組み込む。現状の 5 サービスは liveness +
一覧の read のみで request body を持たないため、本 ADR では枠組み (Error スキーマ +
422 セマンティクス) の用意までに留める。

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
