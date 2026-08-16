# client — フロントエンド (pnpm workspace)

2 つの UI（`app/store` / `app/backoffice`）と、共有パッケージ（`app/api` / `app/ui`）、E2E
（`e2e`）のモノレポ。構成・データ取得・層分け・依存管理の方針は
[doc/frontend.md](../doc/frontend.md)。

## セットアップ

このディレクトリで実行する（`node` は `mise.toml` が固定する）。

```sh
mise install        # node
mise run install    # pnpm install
```

以降のタスク（生成・ビルド・lint・テスト）は `mise tasks` で一覧できる。app ごとのスクリプトは
各 `package.json` を参照。E2E はスタックの起動が要るためリポジトリルートのタスクから実行する
（[e2e/README.md](e2e/README.md)）。
