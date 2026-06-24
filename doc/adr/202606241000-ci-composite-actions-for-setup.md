# ADR-202606241000: CI の setup 系重複ステップをローカル composite action に集約する

- Status: Accepted
- Date: 2026-06-24
- Relates to: ADR-[[202606220716]] (CI をアーキテクチャ量子ごとに分割), [[ci.md]] (CI 規約)

## Context

ADR-[[202606220716]] で CI を量子別 3 ファイル (`ci-customer.yml` / `ci-backoffice.yml` /
`ci-shared.yml`) に分割した結果、setup 系の同一ステップが各ファイル・各ジョブに散らばった。
計測すると重複は次の通り:

- **Node + pnpm セットアップ** (`pnpm/action-setup` + `actions/setup-node` (cache) +
  `pnpm install --frozen-lockfile`): 3 ファイルの全 client 系ジョブで **8 箇所**。
- **Go セットアップ** (`actions/setup-go`, `go-version-file: go.mod`, cache): server-unit /
  server-integration ×2 量子で **4 箇所**。
- **goose の prebuilt 導入** (`curl` + `chmod`): server-integration / e2e ×2 量子で **4 箇所**。
  同一ファイル内は YAML anchor (`&install-goose`) で 1 定義に畳んでいたが、anchor は
  ファイルを跨げない (ADR-[[202606220716]]) ため、customer / backoffice の両ファイルに同じ
  定義を複製していた。

`uses:` は 40 桁 SHA で pin する規約 ([[ci.md]]) のため、action を 1 つ更新するたびに pin を
重複箇所すべてで直す必要があり、Node + pnpm の 8 箇所がとくに保守の負担になっていた。

## Decision

setup 系の重複を **ローカル composite action** (`.github/actions/<name>/action.yml`) に集約し、
各ワークフローからは `uses: ./.github/actions/<name>` で呼ぶ。作成した action は 3 つ:

- `setup-node-pnpm`: pnpm + Node セットアップと frozen install。install は workspace ルート
  (`client`) から実行し app/* と e2e を一括で入れる (pnpm-workspace.yaml)。
- `setup-go`: `go.mod` 追従の Go セットアップ + モジュールキャッシュ。
- `install-goose`: goose の prebuilt バイナリ導入。

決め手と境界:

- **checkout は composite に含めない**。ローカル composite action はリポジトリが checkout 済みで
  ないと解決できないため、各ジョブの先頭に `actions/checkout` を残し、その後に composite を呼ぶ。
- **SHA pin は composite 内に 1 箇所だけ**置く ([[ci.md]] 準拠)。これで action 更新時の修正点が
  Node + pnpm は 8 → 1、Go は 4 → 1 に縮約される。
- **goose は anchor をやめて composite 化**する。anchor がファイルを跨げない制約を composite が
  解消し、customer / backoffice の複製定義が 1 つに集約される。go を使わない e2e ジョブでも
  goose だけ独立に呼べるよう、`setup-go` とは別 action にする。
- **composite 変更時に CI を再実行**させるため、各ワークフローの `paths:` トリガに
  `.github/actions/**` を追加する。native `paths:` で起動制御する ADR-[[202606220716]] の方針上、
  composite を変えても無起動にならないようにする。
- **coverage gate の awk / coverage コメントの sticky action は集約しない**。job ごとに `header` /
  `total` などの入力が異なり、composite 化すると入力受け渡しで抽象度が上がるわりに、SHA pin の
  保守メリットが setup 系ほど大きくないため、現状の inline を維持する。

## Consequences

- action のバージョン更新 (pin 差し替え) が 1 ファイルで済み、setup 重複の保守コストが下がる。
- goose の cross-file 複製が消え、ADR-[[202606220716]] が許容していた「anchor がファイルを跨げない
  ための複製」が 1 箇所に縮約された。
- composite を変更すると `.github/actions/**` トリガで両量子 + shared の CI が再実行される
  (共有インフラなので影響範囲どおり)。
- composite action のステップは job の `defaults.run.working-directory` を継承しない。`setup-node-pnpm`
  の install は working-directory を明示しているため、e2e ジョブ (defaults が `client/e2e`) でも
  workspace ルートから install され挙動は変わらない。
- 量子別ファイルの「読みやすさ」は維持される (ADR-[[202606220716]])。setup の中身は action 名で
  意図が読め、ジョブ本文は各量子固有のステップだけになる。

## Alternatives considered

- **reusable workflow (`workflow_call`) に切り出す**: job 全体を共有するなら有効だが、今回の重複は
  job 内の数ステップ単位。reusable workflow は呼び出し粒度が job で、setup だけの共有には過剰。
  composite action の方がステップ粒度で噛み合う。
- **現状維持 (YAML anchor のみ)**: anchor はファイルを跨げず、量子別 3 ファイル構成
  (ADR-[[202606220716]]) では複製が残る。SHA pin の重複保守も解消できないため不採用。
- **coverage gate / coverage コメントまで composite 化 (最大案)**: 入力 (header/path/total) を持つ
  composite にできるが、setup 系ほど重複保守の利得がなく抽象度だけ上がるため見送り (上記 Decision)。
