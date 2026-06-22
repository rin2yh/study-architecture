# ADR-202606220600: 共有 ui の VRT は store 上の Storybook + test-runner で行う

- Status: Accepted
- Date: 2026-06-22
- Relates to: ADR-[[202606170906]], ADR-[[202606220300]]

## Context

shadcn の UI キットを共有 `ui` パッケージ (`client/app/ui`) に集約し、`store` / `mypage` /
`backoffice` が単一情報源として参照する構成にした (ADR-[[202606170906]] の延長)。`ui` は vendored
shadcn でユニットテストを持たない方針のため、CI では coverage gate を課さず typecheck 専用の
`client-ui` ジョブだけを回している。

デザインシステムを単一情報源にした結果、`ui` の見た目変更は全サービスへ一括で波及する。色・余白・
状態差分といった**視覚的な退行は行カバレッジでは守れない**。スナップショット差分で守る仕組みが要る。

## Decision

- **ツールは Storybook + `@storybook/test-runner` (Playwright)** にする。ローカル/CI 完結で
  スナップショットをリポジトリ内に持て、外部ホスティング (Chromatic 等) の secret・ネットワーク
  許可を増やさずに済む。
- **VRT のホストは `store` アプリに置く** (`一旦 store のみ`)。stories は `store/src/stories/*`
  から `ui/*` を import し、`store` の Vite + Tailwind パイプラインでレンダリングする。Storybook を
  `ui` にコロケートせず `store` 1 つに閉じることで、`mypage` / `backoffice` には手を入れない。
  対象は `button` / `card` / `alert` / `badge` / `label` / `input` / `select` / `separator` /
  `table` / `page-loading`、各コンポーネントを light / `.dark` の両テーマで撮る。
- **ベースラインは固定した Playwright Docker イメージ (`mcr.microsoft.com/playwright:v1.60.0-noble`)
  内で撮り、PNG をリポジトリ内 (`store/__vrt__/__snapshots__`) にコミットする**。フォント/ラスタライズ
  差を排除するため比較・撮影とも常にこのイメージを `container:` で使う。
- **比較 (ゲート) と更新 (撮影) をワークフローで分離する**。CI の `client-store-vrt` は比較専用
  (`--ci`) で、差分か欠落が出たら fail し diff 画像を artifact に上げる (ベースラインは書き換えない)。
  ベースライン更新は専用の `vrt-baseline.yml` が同じイメージで撮り直してコミットする。
  - 理由: 「差分が出たら自動更新」は意図しない退行も受け入れてしまいゲートが無意味になる。更新は必ず
    **人間の承認シグナル**を起点にする。シグナルは PR への `vrt:update` ラベル (PR 中の更新) または
    手動 `workflow_dispatch` (任意ブランチ)。
  - 更新コミットの push が比較ゲートを再発火して HEAD を green にできるよう、push には PAT (`VRT_PAT`)
    を使う。未設定時は `GITHUB_TOKEN` にフォールバックし、その場合は CI を手動再実行する。
  - CI が PR ブランチへ自動 push する初期案は、部分更新不可・ローカルとリモートの乖離・`GITHUB_TOKEN`
    push が CI を再発火せず HEAD が未検証になる、の 3 点で運用が脆く、これを避けるため分離した。
- **依存は `store` の devDeps に直接ピンする** (catalog に入れない)。VRT 専用ツール
  (storybook / @storybook/react-vite / @storybook/test-runner / playwright / jest-image-snapshot /
  http-server / wait-on / concurrently) は他パッケージで共有しないため。
- coverage gate (他 app の 60%) は VRT には課さない。視覚的退行はスナップショット差分で守る分離方針。

## Consequences

- `ui` の見た目変更は PR の `client-store-vrt` で差分として検出でき、意図した変更はベースライン PNG の
  更新を PR に含めることでレビューできる。
- ベースラインが CI イメージ依存になるため、Playwright イメージのタグを上げると再生成が要る。タグは
  SHA 同様に固定 (move-target を避ける) し、上げるときは意図的に再生成する運用とする。
- 手元では Storybook の確認 (`pnpm -F store storybook`) はできるが、ベースライン撮影は CI イメージに
  委ねる。ブラウザバイナリ取得とフォント差でローカル撮影は環境依存になりやすい、という割り切り。
- 更新は承認 (ラベル / dispatch) を起点にする半自動で、`GITHUB_TOKEN` 自動 push の脆さを避けられる。
  代償として最良の体験 (更新後に HEAD が自動 green) には `VRT_PAT` secret の登録が要る。
- `store` に VRT ツールチェーンが乗るぶん install が重くなるが、`client` matrix とは別ジョブのため
  既存の coverage パイプラインには影響しない。

## Alternatives considered

- **Storybook を `ui` にコロケート**: 単一情報源として素直だが、`ui` は build 成果物を出さない方針で
  Vite/Tailwind ホストを持たず、VRT のために `ui` 側へツールチェーンを足すことになる。まずは `store`
  1 つで運用を確かめる (`一旦 store のみ`) ことを優先し、横展開は後続とした。
- **Chromatic (ホスティング + 承認フロー)**: ベースライン管理と UI レビューは手厚いが、外部サービス・
  secret・ネットワーク許可が増える。学習用リポジトリには重い。
- **ベースラインを各自の OS で撮ってコミット**: CI のレンダリング環境と一致せず、フォント差で常時
  diff が出る。撮影を固定イメージ (`vrt-baseline.yml`) に寄せて回避した。
- **CI が比較も更新もして PR ブランチへ自動 push**: 手元に Docker 不要で手軽だが、部分更新不可・
  ローカルとリモートの乖離・HEAD 未検証になり運用が脆い。比較専用 + 専用更新ワークフローに分離した。
- **Playwright screenshot を直接使う (Storybook なし)**: 軽量だが、stories という形でコンポーネントの
  状態カタログを残せる Storybook の資産性を優先した。
