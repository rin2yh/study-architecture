# ADR-202606220716: CI をアーキテクチャ量子 (顧客系 / backoffice) ごとに 2 ワークフローへ分割する

- Status: Accepted
- Date: 2026-06-22
- Relates to: ADR-[[202606170900]] (サービスベースアーキテクチャ), ADR-[[202606170909]] (顧客系/運用系 DB・network 分離), ADR-[[202606211200]] (payment→shipping 非同期 seam), ADR-[[202606221300]] (mypage を store に統合・退役), [[ci.md]] (CI 規約)

## Context

Issue #38 は CI を backend 5 + frontend 3 の **8 量子**ごとに gate する前提だったが、これは量子
(独立デプロイ + 静的結合 + 同期的連結の共有範囲) ではなく**デプロイ単位**の数だった。Step 0 の結合を
計測すると:

- **共有 DB が 2 つ** (ADR-[[202606170909]])。同一 DB = 静的結合 = 同一量子なので、5 サービスは DB
  だけで 2 塊に縮約され 5 量子にならない。
- **UI→サービスは同期直接呼び出し**で、`order` は両 UI から両 DB を跨いで同期に呼ばれる。同期呼び出しは
  同一量子へ引き込む (mypage は ADR-[[202606221300]] で退役済み)。
- **非同期 seam は payment→shipping のみ** (ADR-[[202606211200]])。だが shipping は両 UI から同期到達
  され db-ops も共有するため、この境界だけでは量子を割れない。

厳密には Step 0 は共有 DB + 全面的な同期結合で実質 **1 量子** (サービスベースアーキテクチャの典型)。

## Decision

CI の単位として、ADR-[[202606170909]] の external (顧客) / internal (運用) 境界に寄せた **2 量子**を
採る。境界は edge-proxy (同期中継) と Redis Streams (非同期)。

- **顧客系量子**: store / order / payment / member / db-customer
- **backoffice 量子**: backoffice / product / shipping / db-ops

ワークフローは量子別 2 ファイル (`ci-customer.yml` / `ci-backoffice.yml`) + 量子ではない workspace
共通検査の `ci-shared.yml` の計 3 ファイル。決め手:

- **共有パッケージ ui/api は `ci-shared.yml` で独立検証**。複数 app が参照する共有ライブラリで量子では
  ないため、それ自身の変更時だけ回す。ジョブは ui/api を混ぜない。
- **lint/format は pnpm workspace 全体が対象**で量子にも app にも割れないので、各量子ワークフローに
  `code-quality` を 1 つ置きワークフロー単位で 1 回回す (client matrix では回さない)。ci-shared は
  ui/api 専用で lint を持たない。
- **起動制御は native `paths:`**。`dorny/paths-filter` の動的 matrix も集約 gate も使わない。required
  check を運用しない前提なので、無関係変更で workflow が起動しなくても merge はブロックされず、issue #38
  の「required check 整合」課題が消える。
- **共有 lib の fan-out は両ファイルの `paths:` 複製で表現**。YAML anchor はファイルを跨げないため、量子別
  ファイルでは複製が正攻法。共有 client パッケージ・lockfile の変更は各量子の app ビルドにも影響するので
  量子側トリガにも残す。
- **integration job も量子別に分割**。計測上、顧客系と backoffice 側を相互 import するテストは 0 件、redis
  依存は miniredis (in-process) なので、各量子は自量子の DB だけ起動すればよい。
- **required check は当面運用しない** (学習用リポの費用対効果)。CI は情報提供として回す。

## Consequences

- 変更量子だけが回り、アーキテクチャ境界と CI 構造が一致する (量子の教材としても読みやすい)。
- 共有パスリストを 2 ファイルに複製するため、共有 lib のパス追加時は**両ファイルを揃えて直す**必要がある
  (片方忘れると fan-out が片肺になる)。
- lint/format は各量子の `code-quality` で 1 回ずつ走る。両量子起動時は各々 1 回 (workspace 全体が対象で
  app 単位に割れないための許容コスト)。client matrix での重複実行だった旧構成は解消。
- ui/api は `ci-shared.yml` で個別検証し、api の coverage コメントもここから 1 回だけ出す (旧 client-shared
  のコメント競合は解消)。
- 量子ワークフロー内で server/client が同一トリガを共有するため、client 変更で server job も回る (broad
  fan-out は許容。server/client 分離は保留)。
- 「2 量子」は厳密な量子でなく DB/network 境界 (ADR-[[202606170909]]) に寄せた運用上の近似。edge-proxy 越し
  の同期呼び出しが残り教科書的には単一量子に縮約される点は承知の上。
- **required check を将来有効化する場合**、native `paths:` では無関係変更時に workflow が起動せず check が
  未生成で merge がブロックされうる。その時点で always 起動の集約 gate を別途足す (issue #38 の残課題)。本
  ADR の前提は required を運用しないこと。

## Alternatives considered

- **単一 ci.yml + `dorny/paths-filter` 動的 matrix + 集約 gate** (issue #38 の commit `4fa080a`):
  required を必須運用するなら整合が取れる正攻法。だが required を運用しない本 ADR では、third-party action
  と動的 matrix の複雑さに見合う利得がなく、量子別ファイルの方が構造が素直で学習向き。
- **分割しない (1 量子のまま現行 ci.yml)**: 厳密には正しいが、external/internal の境界は実在し CI を量子に
  揃える学習価値もあるため 2 分割を採る。
- **8 デプロイ単位で分割** (issue #38 の前提): 量子とデプロイ単位を混同し共有 DB を無視するため不採用。
  デプロイ単位の粒度は各量子内の matrix で表現する。
