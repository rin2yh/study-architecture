# ADR-202606221000: `as` 型アサーションを oxlint で禁止する

- Status: Accepted
- Date: 2026-06-22
- Relates to: `client/.oxlintrc.json`、`client/app/api/src/mutator.ts`、PR #21 (store UI / テスト整理) レビュー指摘

## Context

`as` 型アサーションはコンパイラの型チェックを黙らせる。実体と型がずれていても通ってしまい、
runtime のずれを型で守れなくなる。レビュー (PR #21) で、本体・テストの両方に `as` が散在し、
特にテストの `as unknown as ...` が「型を黙らせるためだけ」に積み上がっていることが問題になった。

oxlint には `typescript/consistent-type-assertions` があり、`assertionStyle: "never"` で
`x as T` も角括弧形式 `<T>x` も一律で error にできる (oxlint 1.69.0 でサポート確認)。
`as const` (const assertion) は対象外なので、`{ ok: false as const }` 等の判別用リテラルは残せる。

## Decision

**`client/.oxlintrc.json` に `typescript/consistent-type-assertions: ["error", { assertionStyle: "never" }]`
を入れ、既存の `as` を型注釈・`satisfies`・テストの組み替えで解消する。**

解消の型:

- **本体の `unknown[]` → 具体型**: `JSON.parse` 後の `Array.isArray(parsed)` で `parsed` は
  `any[]` に絞られるため、`as` を外してそのまま返す (cart / checkout の parseItems)。
- **mock 戻り値**: `mockResolvedValue({ ... })` の引数は戻り値型で文脈付けされるので、
  `data`/`status`/`headers` を素直に埋めれば `as Awaited<ReturnType<...>>` は不要。
- **loader/action 引数**: `{ request } as unknown as Parameters<...>[0]` をやめ、`request`/`url`/
  `params`/`pattern`/`context` を持つ実型 (`ServerDataFunctionArgs`) を組んで渡す。
- **route コンポーネントの props 直挿し**: `Comp as unknown as (props) => ...` をやめ、
  `createRoutesStub` の `loader` (loaderData)・throw する loader + `ErrorBoundary` (error)・
  `hydrationData.actionData` (actionData) 経由で実コンポーネントを描画する。
- **DOM 要素の絞り込み**: `getByRole(...) as HTMLButtonElement` は `getByRole<HTMLButtonElement>(...)`。
- **throw された Response の検証**: `(thrown as Response)` は `instanceof Response` で narrow する。

### 例外 (`as` を 1 箇所だけ許可)

`client/app/api/src/mutator.ts` の orval 共通 fetch 実装は `<T>(...): Promise<T>` で、
生成コードが型引数に envelope (`{ data; status; headers }` の判別ユニオン) を渡す。runtime 値
から任意の `T` を**型安全に**構築する手段は無く、`satisfies` でも型注釈でも代替できない。
ここだけ `// oxlint-disable-next-line typescript/consistent-type-assertions` で `as T` を許可し、
理由コメントから本 ADR を参照する。生成コードとの境界に限定した、ただ 1 つのエスケープハッチ。

## Consequences

- 型を黙らせる `as` が CI (oxlint) でブロックされ、以後の混入を防げる。`as const` は使えるまま。
- テストが実 RR の経路 (loader/action/ErrorBoundary) を通るようになり、props を手で捏造する形より
  実挙動に近い。一方 `createRoutesStub` は loader を非同期に解決するため、描画系アサーションは
  `await screen.findBy...` が要る (同期 `getBy...` から書き換えた)。
- mutator の 1 箇所だけ `as` が残る。境界を ADR で明示し、無制限の `as` 復活ではないことを担保する。
- gen 配下は元から lint 対象外 (ADR-[[202606190901]]) なので生成コードは影響を受けない。

## Alternatives considered

- **本体のみ解消・テストは override で除外**: 作業は軽いが、テストの `as unknown as` という
  最も型を殺している層が残る。型安全方針として中途半端なので採らない。
- **severity を `warn`**: CI を落とさず段階移行できるが、warn は放置されやすく `as` が増え続ける。
  全箇所を解消できる規模 (約30) だったので error で確定させる。
- **mutator を非ジェネリックな envelope 返却に変える**: 生成コード側の呼び出し
  (`memberFetch<listMembersResponse>(...)`) が型引数を渡す前提のため、型不整合で typecheck が壊れる。
  生成物を編集しない方針 (ADR-[[202606190901]]) とも反するので不可。inline disable に留める。
