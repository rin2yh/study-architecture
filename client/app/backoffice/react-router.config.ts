import type { Config } from "@react-router/dev/config";

// React Router v7 (旧 Remix 統合) のデフォルトディレクトリ "app/" ではなく既存の src/ を使う。
export default {
  appDirectory: "./src",
  ssr: true,
} satisfies Config;
