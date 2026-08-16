# client — フロントエンド (pnpm workspace)

2 つの UI（`app/store` / `app/backoffice`）と、共有パッケージ（`app/api` / `app/ui`）、E2E
（`e2e`）のモノレポ。構成・データ取得・層分け・依存管理の方針は
[doc/frontend.md](../doc/frontend.md)。

## コマンド

このディレクトリで実行する（`node` は `mise.toml` が固定する）。

```sh
mise install        # node
mise run install    # pnpm install
mise run gen        # api の orval 生成（server/<svc>/api/openapi.yaml → app/api/src/gen/**）
mise run build      # 全 app を vite build + tsc --noEmit
mise run lint       # oxlint + oxfmt --check
mise run test       # vitest

pnpm --filter store dev   # 個別 app の dev サーバ（backoffice も同様）
pnpm format               # oxfmt で整形
```

E2E はスタックの起動が要るためリポジトリルートのタスクから実行する（[e2e/README.md](e2e/README.md)）。
