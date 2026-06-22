# ADR-202606221200: shadcn UI を共有 `ui` ワークスペースパッケージに集約し全 UI で共有する

- Status: Accepted
- Date: 2026-06-22
- Relates to: ADR-[[202606220300]] / ADR-[[202606190901]] / ADR-[[202606170906]] / ADR-[[202606170908]]

## Context

`client/app/{store,mypage,backoffice}` の 3 UI のうち、shadcn の UI キットを持つのは store
だけだった (`store/src/shared/ui/*` + テーマ `styles.css` + `cn` + radix / cva / clsx /
tailwind-merge / tw-animate-css の依存)。mypage / backoffice は素の Tailwind のみで、ボタン・
テーブル・エラー表示をその場で `bg-gray-900` / `border-gray-300` 等の生 class で組んでいた。

このため同じ「ボタン」「テーブル」「エラー表示」がサービスごとに別実装・別トーンになっており、
全社 UI で見た目と挙動を揃える単一情報源が無い。shadcn はコンポーネントを各プロジェクトへ
コピーする配布形態なので、素直に各 app の `shared/ui` へ撒くと実装が 3 つに分裂し、修正のたびに
3 箇所追従する必要が出てドリフトする。

ADR-[[202606220300]] (FSD) では shadcn を各 app の `shared/ui/*` に置く前提だったが、これは
「app 内の層分け」の決定であって「app 間で UI キットをどう共有するか」は対象外だった。

## Decision

shadcn の UI キットを **共有ワークスペースパッケージ `client/app/ui` に集約**し、全 UI が
そこを単一情報源として参照する。`api` パッケージ (ADR-[[202606190901]]) と同じく **ビルド成果物を
出さず TS / CSS ソースを `exports` の subpath で直接公開**し、各 app の Vite / Tailwind が
解決・バンドルする。

- コンポーネントは `ui/src/components/*`、`cn` は `ui/src/lib/utils.ts`、テーマトークンは
  `ui/src/styles/theme.css`。app からは `ui/button` / `ui/table` / `ui/lib` / `@import "ui/styles.css"`
  で参照する。
- パッケージ内のコンポーネントが `cn` を読むときは **相対 import (`../lib/utils`)** にする。
  consumer 側の `@/` エイリアス (app ごとに自分の `src` を指す) に依存させないため。
- Tailwind v4 の自動コンテンツ検出は node_modules を辿らず、`ui` は workspace symlink 経由で
  各 app から読まれる。`ui` コンポーネントの class が purge されないよう、`theme.css` に
  `@source "../components"` を置いて検出対象に明示登録する。
- 各 app の `styles.css` は `@import "ui/styles.css"` の 1 行に縮約し、テーマを全 UI で共有する。
- shadcn の配布形態 (コピー) は維持する。`shared/ui` を全 app に撒く代わりに **置き場所を `ui`
  パッケージ 1 つに寄せた** 形で、CLI で足したコンポーネントもここへ追加する。

FSD (ADR-[[202606220300]]) の `shared` 層の位置づけは保つが、「ドメイン非依存の再利用 UI キット
(shadcn) の置き場」は各 app の `shared/ui` から共有 `ui` パッケージへ移す。app 固有の `shared`
(例: store の `shared/lib/money`) は引き続き各 app に残す。

## Consequences

- ボタン・テーブル・エラー表示・テーマが 1 箇所定義になり、修正が全 UI に一度で反映される。
  実測で mypage / backoffice の生成 CSS は素 Tailwind から約 30KB へ増え、共有テーマ変数
  (`--primary` 等) と共有 `Table` の class (`hover:bg-muted` 等) がコンパイルされることを
  build で確認した。
- store は `shared/ui/*` と `shared/lib/utils` を撤去し `ui/*` 参照へ移行。radix / cva / clsx /
  tailwind-merge / tw-animate-css の直接依存は `ui` パッケージへ移し、共有版数は catalog
  (ADR-[[202606170906]]) に登録して全 app でピン留めした。
- mypage / backoffice はログインフォーム・一覧テーブル・ローダ・エラー境界を共有
  コンポーネント (`Button` / `Input` / `Label` / `Table` / `Alert` / `PageLoading`) に置換し、
  生 `gray-*` / `red-*` をテーマトークンへ寄せた。既存テスト (`role="status"` / `role="alert"` /
  ボタン名 / セル文言) は不変で全 74 件 green。
- トレードオフ: shadcn CLI (`npx shadcn add`) は `@/lib/utils` 形式の import を生成するため、
  `ui` パッケージへ追加した直後に相対 import へ書き換える手間が要る (`ui/README.md` に明記)。
  ソース共有モデルと CLI の `@/` 前提が噛み合わないことの代償で、相対 import の堅牢さを優先した。
- FSD の「`shared/ui` に shadcn」前提が変わるため、ADR-[[202606220300]] の該当部分は本 ADR で
  更新する (本文・結論は履歴として残す)。コンポーネント props 規約の vendored shadcn 例外の
  パス参照も `ui` パッケージへ更新した。

## Alternatives considered

- **各 app の `shared/ui` に shadcn をコピーして撒く (FSD 素直案)**: FSD の層規約には最も忠実
  だが、同一コンポーネントが 3 実装に分裂し修正のたびに 3 箇所追従が要る。「一貫したデザイン
  システム」という動機 (単一情報源) を満たさず却下。
- **`ui` をビルドして dist を配る (通常の npm ライブラリ形態)**: 型・CSS の出力管理と watch が
  要り、モノレポ内 1 リポジトリでは過剰。`api` が既にソース直公開で回っている前例に倣い却下。
- **`cn` をパッケージ自己参照 (`ui/lib`) で読む**: CLI 生成物との相性は相対より良いが、
  パッケージのリンク状態に解決を依存させ脆い。パッケージ内は相対 import が定石なので相対を採用。
- **テーマ CSS を各 app に複製したまま共有はコンポーネントだけ**: トークンがドリフトし配色が
  サービス間でずれる。テーマこそ揃えたい中核なので `theme.css` ごと共有して却下。
