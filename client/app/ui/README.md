# ui

全サービス UI 共通の shadcn デザインシステム。`store` / `mypage` / `backoffice` から
ワークスペースパッケージとして参照する単一情報源。

## 使い方

各 app の `package.json` に `"ui": "workspace:*"` を追加し、CSS とコンポーネントを読み込む。
CSS は各 app のエントリ (`src/root.tsx`) で直接 import する。

```tsx
import "ui/styles.css";
```

```tsx
import { Button } from "ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "ui/table";
import { cn } from "ui/lib";
```

ビルド成果物は出さず TS/CSS ソースを `exports` で直接公開する (`api` パッケージと同方針、
ADR-202606190901)。各 app の Vite/Tailwind がソースを解決・バンドルする。

## 構成

- `src/components/*` — shadcn の UI キット + 共有 `page-loading`。`cn` は `../lib/utils`
  からの相対 import で参照する (consumer 側の `@/` エイリアスに依存しないため)。
- `src/lib/utils.ts` — `cn`。
- `src/styles/theme.css` — Tailwind v4 のテーマトークン (`:root` / `.dark` / `@theme inline`)
  と base レイヤー。`@source "../components"` で自パッケージのコンポーネントを Tailwind の
  コンテンツ検出に登録する。

## shadcn CLI で追加する場合

`npx shadcn@latest add <name>` は `@/lib/utils` 形式の import を生成するため、追加後に
`cn` の import を `../lib/utils` (相対) へ書き換える。`exports` にも subpath を追記する。

## 見た目の退行検知 (VRT)

このパッケージのコンポーネントは `store` のページ VRT 経由で実合成として撮られ、退行が検出される
(コンポーネント単体の VRT は持たない)。運用は [`../store/README.md`](../store/README.md) と
ADR-202606220600 を参照。
