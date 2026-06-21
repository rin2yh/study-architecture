# ADR-202606211300: UI コンポーネントは shadcn/ui (Radix + Tailwind) を取り込み式で持つ

- Status: Accepted
- Date: 2026-06-21

## Context

store の画面は Tailwind ユーティリティの直書きで組んでおり、ボタン・カード・セレクト・アラート等
の見た目と挙動（特にアクセシブルな Select）が各所で重複・自前実装になりかけていた。共通の UI
部品群が要る。既存スタックは React 19 + Tailwind CSS v4 + React Router v7 (SSR) + pnpm モノレポで、
catalog には既に `lucide-react` がある。

## Decision

**shadcn/ui**（Radix UI プリミティブ + Tailwind、`lucide-react` アイコン）を採用する。部品は npm
依存としてブラックボックスに持つのではなく、**`src/components/ui/` に取り込んでリポジトリで所有**
する（shadcn の標準方式）。`cn`（clsx + tailwind-merge）を `@/lib/utils` に置く。

導入範囲は段階的にし、**まず store だけ**に入れて cart / checkout / home の導線を置き換える。
共有 UI パッケージ化（`@ec/api` のような横断パッケージ）は、mypage / backoffice が同じ部品を欲し
始めた時点で別途行う。

## Consequences

- Tailwind v4 をそのまま活かせ、スタイル体系が二重化しない。Radix ベースで a11y（フォーカス・
  キーボード・ARIA）が部品側に入る。
- **ランタイム CSS-in-JS を持ち込まない**ため、RR v7 の SSR でスタイル抽出やハイドレーションの
  追加設定が要らない。
- 部品コードを所有するので将来の改変が自由な反面、上流の更新は手動取り込みになる。
- vitest は vite の dev plugin を読まないため、`@` エイリアスを `vitest.config.ts` の
  `resolve.alias` にも与える必要がある（tsconfig paths と二重定義になる）。

## Alternatives considered

- **Mantine**: 全部入りで DX は良いが独自スタイル体系（CSS Modules / PostCSS）を持ち、Tailwind と
  二重化する。
- **MUI**: 部品は豊富だが emotion ランタイム CSS-in-JS で SSR 設定が重く、Material 固定。
- **daisyUI**: 純 Tailwind プラグインで最軽量だが、modal / combobox 等の対話部品は結局 headless
  JS が別途必要で、Select のような a11y 部品を賄えない。
