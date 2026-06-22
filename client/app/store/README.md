# store

顧客向けストア (product / order / payment / member)。共有 `ui` パッケージを参照する。

## Visual Regression Test (VRT)

共有 `ui` の見た目を Storybook + `@storybook/test-runner` (Playwright) で固定する。stories は
`src/stories/*.stories.tsx` から `ui/*` を import し、各コンポーネントを light / `.dark` の両テーマで
撮ってスナップショット差分で退行を検出する。方針は ADR-[[202606220600]]。

```sh
pnpm -F store storybook        # 手元で Storybook を開いて確認する
pnpm -F store build-storybook  # storybook-static を生成する (VRT の前段)
```

### CI は比較専用

`client-store-vrt` ジョブはコミット済みベースライン (`__vrt__/__snapshots__/`) と**比較するだけ**で、
差分か欠落が出たら fail し diff 画像を `vrt-diff` artifact に上げる。CI はベースラインを書き換えない。

### ベースラインの更新は専用ワークフロー (`vrt-baseline.yml`)

ベースラインはレンダリング環境に依存するため、比較ゲートと**同じ固定 Playwright イメージ**で撮り直す
必要がある。手元の OS で撮ると差分が出るので、撮影は CI に寄せて以下の承認トリガで回す。

- **PR で更新する**: 見た目を意図的に変えた PR に **`vrt:update` ラベル**を付ける。`vrt-baseline.yml`
  が同じイメージで全ベースラインを撮り直し、その PR ブランチへコミットする。再更新したいときはラベルを
  一度外して付け直す。
- **任意のブランチで更新する**: Actions から `VRT baseline` を **手動 dispatch** し、対象ブランチを
  指定する (`workflow_dispatch` は main にこのワークフローが入って以降に使える)。
- **退行を確認する**: 差分は `vrt-diff` artifact で確認する。意図した変更なら上のトリガで更新し、
  そうでなければコードを直す。

更新コミットの push が比較ゲートを再発火して HEAD を green にするには、`repo` 権限を持つ PAT を
リポジトリ secret **`VRT_PAT`** に登録する。未登録でもベースラインは更新されるが、その場合は CI を
手動で再実行する。

CI で使う Playwright イメージのタグを上げたときも、このワークフローで全ベースラインを撮り直す。
