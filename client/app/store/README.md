# store

顧客向けストア (product / order / payment / member)。共有 `ui` パッケージを参照する。

## Visual Regression Test (VRT)

store の**ページ**の見た目を Storybook + `@storybook/test-runner` (Playwright) で固定する。stories は
`src/stories/*.stories.tsx` で各ページ (home / cart / checkout) を mock データで描画し、light / `.dark`
の両テーマで撮ってスナップショット差分で退行を検出する。実ページのローダはバックエンドを叩くため、
stories はメモリルータ + fixtures でページ合成だけを再現する。方針は ADR-[[202606220600]]。

```sh
pnpm -F store storybook        # 手元で Storybook を開いて確認する
pnpm -F store build-storybook  # storybook-static を生成する (VRT の前段)
```

### ベースラインは main の artifact (`vrt.yml`)

ベースライン PNG はリポジトリに置かず、**main の VRT 実行の artifact (`vrt-baselines`)** に持つ。
`ubuntu-24.04` 上で `playwright install` した Chromium で撮る。store ページ / `ui` に関係ない変更では
走らないよう `paths` で絞っている。

- **PR**: main の最新 `vrt-baselines` を取得して比較し、差分が出たら fail、diff 画像を `vrt-diff`
  artifact に上げる。main にベースライン未確立のときは比較せず gate を通す。
- **main へ merge**: 撮り直して `vrt-baselines` artifact を更新する。

### 見た目を意図的に変えるとき

PR では main のベースラインと比較するため、意図した変更でも `client-store-vrt` は **fail** する
(差分が `vrt-diff` artifact に出る)。差分を確認のうえ merge すれば、main 側の実行がベースラインを
更新する。PR 上でベースラインを「承認して green にする」運用は持たない。
