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

### ベースラインの更新

ベースライン PNG (`__vrt__/__snapshots__/`) は**固定した Playwright イメージ上の CI で生成・コミット**
する。レンダリング環境が一致しないと差分が出るため、手元では更新しない。

- **意図した見た目変更を取り込む**: 対象コンポーネントのベースライン PNG を削除して push する。
  `client-store-vrt` ジョブがベースライン未コミットを検知し、同じ Playwright イメージ上で生成して
  PR ブランチへコミットする (`[skip ci]`)。
- **退行を確認する**: 差分が出たジョブは fail し、diff 画像を `vrt-diff` artifact に上げる。意図した
  変更なら上記の手順でベースラインを更新し、そうでなければコードを直す。

CI で使う Playwright イメージのタグを上げたときは、全ベースラインを消して再生成する。
