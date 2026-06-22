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
  内で生成・比較する**。フォント/ラスタライズ差を排除するため、CI もこのイメージを `container:` に
  指定する。ベースライン (PNG) はリポジトリ内 (`store/__vrt__/__snapshots__`) にコミットする。
- **ベースラインの生成主体は CI とする**。手元 (各自の OS) で撮ると CI のイメージと環境が一致せず
  差分が出るため、ベースラインは「対象 PNG を消して push → CI が同一イメージ上で生成・コミット」する
  フローに一本化する。`client-store-vrt` ジョブはベースライン未コミット時のみ初回生成分を PR ブランチへ
  push し、以後は比較専用 (`--ci`) で差分が出たら fail し diff 画像を artifact に上げる。
- **依存は `store` の devDeps に直接ピンする** (catalog に入れない)。VRT 専用ツール
  (storybook / @storybook/react-vite / @storybook/test-runner / playwright / jest-image-snapshot /
  http-server / wait-on / concurrently) は他パッケージで共有しないため。
- coverage gate (他 app の 60%) は VRT には課さない。視覚的退行はスナップショット差分で守る分離方針。

## Consequences

- `ui` の見た目変更は PR の `client-store-vrt` で差分として検出でき、意図した変更はベースライン PNG の
  更新を PR に含めることでレビューできる。
- ベースラインが CI イメージ依存になるため、Playwright イメージのタグを上げると再生成が要る。タグは
  SHA 同様に固定 (move-target を避ける) し、上げるときは意図的に再生成する運用とする。
- 手元では Storybook の確認 (`pnpm -F store storybook`) はできるが、ベースライン更新は CI に委ねる。
  ブラウザバイナリの取得が要るためローカルの撮影は環境依存になりやすい、という割り切り。
- `store` に VRT ツールチェーンが乗るぶん install が重くなるが、`client` matrix とは別ジョブのため
  既存の coverage パイプラインには影響しない。

## Alternatives considered

- **Storybook を `ui` にコロケート**: 単一情報源として素直だが、`ui` は build 成果物を出さない方針で
  Vite/Tailwind ホストを持たず、VRT のために `ui` 側へツールチェーンを足すことになる。まずは `store`
  1 つで運用を確かめる (`一旦 store のみ`) ことを優先し、横展開は後続とした。
- **Chromatic (ホスティング + 承認フロー)**: ベースライン管理と UI レビューは手厚いが、外部サービス・
  secret・ネットワーク許可が増える。学習用リポジトリには重い。
- **ベースラインを手元で生成しコミット**: CI のレンダリング環境と一致せず、フォント差で常時 diff が
  出る。固定イメージ内生成に一本化して回避した。
- **Playwright screenshot を直接使う (Storybook なし)**: 軽量だが、stories という形でコンポーネントの
  状態カタログを残せる Storybook の資産性を優先した。
