# ui

全サービス UI 共通の shadcn デザインシステム。`store` / `backoffice` から
ワークスペースパッケージとして参照する単一情報源。パッケージの構成と shadcn CLI で
コンポーネントを追加するときの注意は [doc/frontend.md](../../../doc/frontend.md)。

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
