---
paths:
  - ".github/workflows/**"
---

# CI 規約

`.github/workflows/` 配下を編集するときに守る方針。

- **CI で `mise` を使わない**
  - GitHub Actions の job では `actions/setup-go` / `actions/setup-node` / `pnpm/action-setup` を直接呼び、ローカル開発用の `mise.toml` を再利用しない
  - 理由: mise はローカルのツール固定が役割。CI から参照すると `mise install` のキャッシュ・マトリクス・既存 Actions エコシステムと噛み合わず、setup 時間と保守の負担が増える
  - 環境変数 (`CGO_ENABLED=0` 等) は workflow の `env:` で明示する

- **`uses:` は最新版 + 40 桁 SHA で pin する**
  - 形式: `uses: <org>/<action>@<full-40-char-sha>  # vX.Y.Z`
  - 理由: タグは可変・SHA は不変。GitHub Actions 自身の supply-chain ベストプラクティスでもある
  - 取得手段: `git ls-remote --tags https://github.com/<owner>/<repo> refs/tags/<tag>^{}` か release page
  - 新 action を追加するとき・既存 action を更新するときも常に pin する

- **`runs-on:` は move-target を避けてバージョン明記**
  - `ubuntu-latest` ではなく `ubuntu-24.04` のような明示版を使う
  - 理由: `latest` が突然上がるとビルドが何もしてないのに壊れる。新 LTS が出たタイミングで明示更新する運用の方が予測可能

- **`actions/checkout` は git push しない job で `persist-credentials: false`**
  - 後段で `git push` や認証付き `gh` を使わないジョブは default の credential 残置をやめる
  - 理由: 残置された `extraheader` の token が後続 step の事故 (誤 push、third-party action による盗用) の経路になる。push する job だけ default (true) のままにする

関連: [naming-and-pnpm-conventions の pnpm ポリシー](../../client/pnpm-workspace.yaml)、本ファイル変更は `[ADR 0011]` も参照すると意図が掴みやすい。
