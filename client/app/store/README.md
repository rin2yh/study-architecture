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

### CI は比較専用 (`vrt.yml`)

`client-store-vrt` ジョブはコミット済みベースライン (`__vrt__/__snapshots__/`) と**比較するだけ**で、
差分か欠落が出たら fail し diff 画像を `vrt-diff` artifact に上げる。CI はベースラインを書き換えない。
`ubuntu-24.04` 上で `playwright install` した Chromium を使う。store ページ / `ui` に関係ない変更では
走らないよう `paths` で絞っている。

### ベースラインの更新は専用ワークフロー (`vrt-baseline.yml`)

ベースラインはレンダリング環境に依存するため、比較ゲートと**同じ runner (`ubuntu-24.04` + Playwright)**
で撮り直す必要がある。手元の OS で撮ると差分が出るので、撮影は CI に寄せて承認トリガで回す。

- **更新する**: 見た目を意図的に変えた PR に **`vrt:update` ラベル**を付ける。`vrt-baseline.yml` が
  全ベースラインを撮り直して PR ブランチへコミットする。再更新したいときはラベルを一度外して付け直す。
- **退行を確認する**: 差分は `vrt-diff` artifact で確認する。意図した変更なら上のラベルで更新し、
  そうでなければコードを直す。

ラベルでの更新コミットは `GITHUB_TOKEN` の push なので比較ゲートは再発火しない。HEAD を green にするには
更新後に何か push する (次のコミットで比較が回る)。
