# ADR-202606220716: CI をアーキテクチャ量子 (顧客系 / backoffice) ごとに 2 ワークフローへ分割する

- Status: Accepted
- Date: 2026-06-22
- Relates to: ADR-[[202606170900]] (サービスベースアーキテクチャ), ADR-[[202606170909]] (顧客系/運用系 DB・network 分離), ADR-[[202606211200]] (payment→shipping 非同期 seam), [[ci.md]] (CI 規約)

## Context

Issue #38 は CI を「デプロイ量子 (独立デプロイ単位)」ごとに gate したいとし、backend 5
(product/order/payment/member/shipping) + frontend 3 (store/mypage/backoffice) の **8 量子**を
前提にしていた。だが計測すると、これはアーキテクチャ量子ではなく**個別デプロイ可能な成果物
(デプロイ単位) の数**であり、量子の定義 (独立デプロイ + 静的結合 + 同期的連結を共有する範囲。
非同期境界が量子を割る) とずれていた。

Step 0 の結合を計測した結果:

- **共有 DB が 2 つ** (ADR-[[202606170909]]): `db-customer`={order, payment, member} /
  `db-ops`={product, shipping}。同一 DB = 静的結合 = 同一量子なので、5 サービスは DB だけで
  既に 2 つの塊に縮約され、5 量子にはならない。
- **UI→サービスは同期直接呼び出し**: store→{product, order, payment, member}、
  mypage→{member, order, shipping}、backoffice→{product, order, shipping}。`order` は 3 UI 全部
  から、サービスも両 DB を跨いで同期で呼ばれる。同期呼び出しは同一量子へ引き込む。
- **唯一の非同期 seam** は payment→shipping (Redis Streams, ADR-[[202606211200]])。ただし
  shipping は mypage/backoffice から同期到達され product と db-ops も共有するので、この境界
  だけでは量子を割り切れない。

厳密な教科書的読みでは Step 0 は実質 **1 量子** (共有 DB + 全面的な同期結合) で、これは
サービスベースアーキテクチャの典型 (共有 DB ゆえ単一量子) である。

## Decision

CI を組む単位として、ADR-[[202606170909]] の **external (顧客) / internal (運用)** 境界に寄せた
**2 量子**を採る。境界は edge-proxy (同期中継) と Redis Streams (payment→shipping, 非同期)。

- **顧客系量子**: `store` / `mypage` / `order` / `payment` / `member` / `db-customer`
- **backoffice 量子**: `backoffice` / `product` / `shipping` / `db-ops`

ワークフローは量子ごとの 2 ファイル (`ci-customer.yml` / `ci-backoffice.yml`) に加え、量子では
ない workspace 共通検査を持つ `ci-shared.yml` の計 3 ファイルにする。

- **共有パッケージ ui/api 専用の `ci-shared.yml`** を置く。ui/api は複数 app が参照する量子では
  ない共有ライブラリで、それ自身の変更時だけ独立に検証する (`client/app/ui` / `client/app/api` と
  共有依存の変更で起動)。ジョブは ui と api を混ぜず `client-ui` / `client-api` に分ける。
- **lint/format (code quality) は単一 pnpm workspace 全体が対象**で量子でも app でも割れないため、
  各量子ワークフロー (customer / backoffice) に `code-quality` ジョブを 1 つ置き、ワークフローごとに
  1 回回す (client matrix の各 app では回さない)。ci-shared は ui/api 専用なので lint は持たない
  (ui/api の変更は量子側の `code-quality` でも lint される)。per-app の typecheck/coverage/build は
  各量子ワークフロー側。
- **起動制御は native `paths:`** で行い、`dorny/paths-filter` の動的 matrix も集約 gate job も
  使わない。ブランチ保護で個別 check を必須にしない運用 (下記) のため、無関係変更で workflow
  自体が起動しなくても merge はブロックされず、issue #38 の「required check 整合」課題が消える。
- **共有 lib の fan-out** は、共有パス (`server/internal` / `server/tools` / `go.mod` / `go.sum` /
  `client/app/ui` / `client/app/api` / lockfile・workspace・`package.json` / `tsconfig.json` /
  `scripts` / `compose.yaml`) を**両ファイルの `paths:` に複製**して表現する。YAML anchor は
  ファイルを跨げないため、量子別ファイルでは複製が正攻法になる。共有 client パッケージ (ui/api)・
  lockfile の変更は各量子の app ビルドにも影響するので、量子ワークフローのトリガにも残し、
  `ci-shared.yml` と合わせて影響範囲をすべて再実行する。
- **integration job も量子別に分割**する。計測上、顧客系サービスと backoffice 側サービスを相互
  import するテストは 0 件で、redis 依存テストは miniredis (in-process) のため、各量子の
  integration は自量子の DB だけ起動すればよい。
- **ブランチ保護の required check は当面運用しない** (学習用リポの費用対効果から)。CI は情報提供
  として回す。

## Consequences

- CI が変更量子だけを回すようになり、無関係な量子の job が走らない。アーキテクチャ境界と CI の
  構造が一致し、量子の概念を学ぶ教材としても読みやすい。
- 共有パスリストを 2 ファイルに複製するため、共有 lib のパス追加時は**両ファイルを揃えて直す**
  必要がある (片方忘れると fan-out が片肺になる)。
- lint/format は各量子ワークフローの `code-quality` ジョブで 1 回ずつ走る。client matrix の app
  ごとに重複実行していた旧構成は解消した。両量子が起動した変更では各々 1 回ずつ走る (workspace
  全体が対象で app 単位に割れないための許容コスト)。
- 共有パッケージ ui/api は `ci-shared.yml` の `client-ui` / `client-api` で個別に検証し、api の
  coverage コメントもここから 1 回だけ出す (旧 client-shared のコメント競合は解消)。量子側は
  ui/api 変更で app ビルドへの影響を再検証する。
- 量子ワークフロー内では server/client が同一トリガを共有するため、client 変更で server job も回る
  (broad fan-out は許容。server/client のワークフロー分離は保留)。
- **「2 量子」は厳密な量子ではなく、DB/network 境界 (ADR-[[202606170909]]) に寄せた運用上の近似**
  である。edge-proxy 越しの同期呼び出しが残るため、教科書的には単一量子に縮約される点は承知の上。
- **required check を将来有効化する場合**、native `paths:` では無関係変更時に workflow が起動せず
  check が未生成になり merge がブロックされうる。その時点で always 起動の集約 gate を別途足す
  (issue #38 の残課題) 必要がある。本 ADR の前提は「required を運用しない」こと。

## Alternatives considered

- **単一 ci.yml + `dorny/paths-filter` 動的 matrix + 集約 gate (issue #38 の commit `4fa080a`)**:
  required check を必須運用するなら集約 gate で整合が取れる正攻法。だが本 ADR は required を運用
  しないため、third-party action と動的 matrix の複雑さに見合う利得がない。量子別ファイルの方が
  構造が素直で学習向き。
- **分割しない (1 量子のまま現行 ci.yml)**: 厳密には正しいが、external/internal の境界は実在し
  CI を量子に揃える学習価値もあるため、2 分割を採る。
- **8 デプロイ単位で分割**: issue #38 の前提。量子とデプロイ単位を混同しており、共有 DB を無視
  すると量子の理解を誤るため不採用。デプロイ単位の粒度は各量子内の matrix で表現する。
