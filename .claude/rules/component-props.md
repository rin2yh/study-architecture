---
paths:
  - "client/**/*.tsx"
---

# コンポーネント props 規約

React コンポーネントの props は、シグネチャ内のインライン型リテラルではなく、**コンポーネント
直前に名前付き `interface`（`<コンポーネント名>Props`）で事前定義**し、シグネチャからは
それを参照する。

- ✗ インライン型リテラル

  ```tsx
  export function ProductRow({ product, cart }: { product: Product; cart: Cart }) { ... }
  ```

- ✓ 事前定義した interface を参照

  ```tsx
  interface ProductRowProps {
    product: Product;
    cart: Cart;
  }

  export function ProductRow({ product, cart }: ProductRowProps) { ... }
  ```

理由: props 形状に名前が付くことで参照・再利用ができ、宣言（型）と実装（関数）が分離されて
シグネチャ行が短くなり見通しが良くなる。インライン型は形状が育つほどシグネチャを圧迫し、
差分も読みにくくなる。

例外（インラインのままでよい / interface 化しない）:

- React Router の生成型（`Route.ComponentProps` / `Route.ErrorBoundaryProps` /
  `Route.ActionArgs` など）。フレームワークが供給する型なので独自 interface に置き換えない。
- vendored な shadcn UI（共有 `ui` パッケージの `app/ui/src/components/*`）。
  `React.ComponentProps<...>` を使う生成物の作法に従い、手を入れない。
- props を取らないコンポーネント（定義するものが無い）。
