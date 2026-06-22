# ADR-202606220600: store のページ VRT を Storybook + test-runner で行う

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

ただし**コンポーネント単体の VRT はデザインシステム (`ui`) の責務**であり、`store` が持つべきは
**自分のページ**の見た目である。ページ VRT は home / cart / checkout を実合成で撮るので、結果的に
共有 `ui` の退行も実利用の文脈で検出できる。まずは `store` のページのみを対象にする (`一旦 store のみ`)。

## Decision

- **ツールは Storybook + `@storybook/test-runner` (Playwright)** にする。ローカル/CI 完結で
  スナップショットをリポジトリ内に持て、外部ホスティング (Chromatic 等) の secret・ネットワーク
  許可を増やさずに済む。
- **対象はコンポーネント単体ではなく `store` のページ**。stories (`store/src/stories/*`) は home /
  cart / checkout の各状態 (一覧/空、カート有/空、フォーム/エラー/確定/空) を fixtures + メモリルータで
  描画し、light / `.dark` の両テーマで撮る。実ページのローダはバックエンドを叩くため、stories は
  ページ合成だけを再現する。`mypage` / `backoffice` には手を入れない。
- **撮影は `ubuntu-24.04` runner 上で `playwright install` した Chromium で行う**。比較・撮影とも同じ
  runner なのでレンダリング差は出ない。Docker コンテナは使わない (plain runner で足りる)。
- **ベースラインはリポジトリにコミットせず、main の VRT 実行の artifact (`vrt-baselines`) に持つ**。
  単一の `vrt.yml` がイベントで振る舞いを変える:
  - **PR**: main の最新 `vrt-baselines` を取得して比較 (`--ci`)。差分が出たら fail し diff を artifact に
    上げる。main 未確立時は比較せず gate を通す。
  - **main へ push**: 撮り直して `vrt-baselines` artifact を更新する (= merge 時にベースライン更新)。
  - 理由: PNG をリポジトリに置かず、CI からの push もなくす。意図した変更は「PR で fail → 差分確認 →
    merge → main がベースライン更新」で取り込む。PR 上でベースラインを承認して green にする運用は持たない。
  - CI が比較も更新も兼ねて PR ブランチへ自動 push する初期案は、部分更新不可・ローカルとリモートの
    乖離・HEAD が未検証、の 3 点で運用が脆かった。artifact 方式はこれらを丸ごと回避する。
- **依存は `store` の devDeps に直接ピンする** (catalog に入れない)。VRT 専用ツール
  (storybook / @storybook/react-vite / @storybook/test-runner / playwright / jest-image-snapshot /
  http-server / wait-on) は他パッケージで共有しないため。serve + 実行はワークフローの shell で
  組むので、package.json には `storybook` / `build-storybook` だけ置く。
- coverage gate (他 app の 60%) は VRT には課さない。視覚的退行はスナップショット差分で守る分離方針。

## Consequences

- PNG をリポジトリにコミットしないので git 履歴が汚れず、CI からの push も不要 (secret/PAT も不要)。
  ベースライン更新は main への merge で自動的に行われる。
- **意図した見た目変更は PR では必ず fail する** (main のベースラインと違うため)。差分は `vrt-diff`
  artifact で確認し、納得して merge する運用。PR 上で「承認して green」にはできない、という割り切り。
- artifact の保持期間 (90 日) を超えて main の store/`ui` が変わらないとベースラインが失効する。その間の
  PR は比較スキップ (gate を通す) になる。次の main 変更で再生成される。
- ベースラインが `ubuntu-24.04` runner のレンダリング依存のため、GitHub が runner イメージのフォント等を
  更新すると差分が出うる。次の main push で更新される (Docker 固定より緩い代わりに構成は単純)。
- `store` に VRT ツールチェーンが乗るぶん install が重くなるが、別ワークフローのため既存の coverage
  パイプラインには影響しない。

## Alternatives considered

- **コンポーネント単体 (`button` / `card` …) を `store` で VRT**: 当初こうしたが、コンポーネント単体の
  見た目はデザインシステム (`ui`) の責務で `store` の責務ではない。`store` は自分のページを撮るべき、と
  整理してページ VRT に切り替えた (`ui` 単体 VRT は後続)。
- **Storybook を `ui` にコロケート**: 単一情報源として素直だが、`ui` は build 成果物を出さない方針で
  Vite/Tailwind ホストを持たず、VRT のために `ui` 側へツールチェーンを足すことになる。`store` のページを
  撮る今回の目的では `store` に置くのが自然。
- **固定 Playwright Docker イメージを `container:` で使う**: フォント差を最も厳密に固定できるが、
  `ubuntu-24.04` runner + `playwright install` で十分回る。構成を単純にするためコンテナは使わない。
- **Chromatic (ホスティング + 承認フロー)**: ベースライン管理と UI レビューは手厚いが、外部サービス・
  secret・ネットワーク許可が増える。学習用リポジトリには重い。
- **ベースラインを各自の OS で撮ってコミット**: CI のレンダリング環境と一致せず、フォント差で常時
  diff が出る。撮影を固定イメージ (`vrt-baseline.yml`) に寄せて回避した。
- **CI が比較も更新もして PR ブランチへ自動 push**: 手元に Docker 不要で手軽だが、部分更新不可・
  ローカルとリモートの乖離・HEAD 未検証になり運用が脆い。比較専用 + 専用更新ワークフローに分離した。
- **Playwright screenshot を直接使う (Storybook なし)**: 軽量だが、stories という形でコンポーネントの
  状態カタログを残せる Storybook の資産性を優先した。
