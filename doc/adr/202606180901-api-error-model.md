# ADR-202606180901: API エラーモデルを共通 Error スキーマ + ErrorJSON ミドルウェアに集約する

- Status: Accepted
- Date: 2026-06-18
- Related: ADR-[[202606170901]] (codegen-first), ADR-[[202606170907]] (Gin)

## Context

Step 0 のエラー整形は `ErrorJSON()` の 400 / 500 の 2 値だけで、OpenAPI にエラー定義が無かった (issue #2)。このため body 型がクライアント (orval) に伝わらず、404 / 409 / 422 のようなドメインのセマンティクスも表現できなかった。

## Decision

- **契約を OpenAPI に置く**: 5 サービス共通の `Error` スキーマ (`code` / `message`) と再利用可能な `default` レスポンスを定義する。oapi-codegen / orval が同じ契約から型を生成し、サーバ/クライアントで共有する (ADR-[[202606170901]])。
- **status↔code のマッピングは `ErrorJSON` 1 箇所に集約**する (ADR-[[202606170907]])。handler は `c.Error(err)` で積むだけ。整形・隠蔽・ログレベル判断はミドルウェアに寄せ、隠蔽漏れと shape のばらつきを防ぐ。
- **ドメインのセマンティクスは型付き `middleware.AppError` で表現**する。404 / 409 / 422 を出すかは handler が `AppError` を選ぶだけにし、status と code の対応は `ErrorJSON` を見れば分かる形にする。
- **`code` は string にして status と緩く対応づける**。同じ 409 でも理由が複数あり得るため、後から細分化できる余地を残す。
- `AppError` 以外・非 Public なエラーは従来どおり内部詳細を隠して 500 を返す。明示的に組み立てた `AppError` だけが文言を透過する。

`code` の値・status マッピング・`AppError` コンストラクタ群はソース (`server/internal/middleware`) を SSOT とし、OpenAPI は `code: string` だけ定義して値を列挙しない (一覧を yaml や本 ADR に写すと追加のたび全箇所を直す羽目になり、`NewError` の任意コードも拾えない)。

入力検証の段でこのモデルが実際に使われ、机上の status にならない: 構文不正は gin binding → 400、業務的に不正 (金額が負など) は 422、unique 違反 (SQLSTATE 23505) は 409、未存在は 404。db 固有エラーの 409 / 404 への正規化は共通 `server/internal/dberr` に寄せ、pgx 依存を handler に漏らさず 5 サービスで使い回す。

## Consequences

- エラー body の形が生成型としてサーバ/クライアントに伝播し、クライアントは `default` レスポンスでエラーを型安全に扱える。
- 404 / 409 / 422 の発行点が handler、status↔code の対応が `ErrorJSON` の 1 箇所に分かれ、追跡しやすい。
- `default` レスポンス追加で orval が `HTTPStatusCode*` 型を tag ごとに生成し、tags-split で重複 (TS2308) が出る。サービス別 barrel (`src/<svc>.ts`) の明示 re-export で解消している。

## Alternatives considered

- **handler ごとに status/JSON を直書き**: 隠蔽漏れや shape のばらつき、横展開コストが上がる。
- **code を HTTP status と 1:1 固定**: 同一 status に複数理由があり得るため細分化の余地を潰す。
- **`includeHttpResponseReturnType: false` でレスポンス型を素の data に倒す**: `HTTPStatusCode*` を避けられるが、既存の `{ data, status, headers }` 契約 (mutator / UI ローダが依存) を壊すため見送り。
