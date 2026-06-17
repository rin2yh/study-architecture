import { defineConfig } from "vitest/config";
import viteReact from "@vitejs/plugin-react";

// テスト専用設定。TanStack Start / nitro プラグインは SSR 用で
// vitest 実行を妨げるため読み込まず、JSX 変換に必要な react プラグインのみ使う。
export default defineConfig({
  plugins: [viteReact()],
  test: {
    environment: "jsdom",
    globals: true,
    include: ["src/**/*.test.{ts,tsx}"],
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary"],
      reportsDirectory: "./coverage",
      // 手書きコードのみを分母に含める
      include: ["src/routes/index.tsx"],
      // 生成・設定・シェルは除外
      exclude: [
        "**/*.gen.*",
        "src/routeTree.gen.ts",
        "**/*.config.*",
        "src/router.tsx",
        "src/routes/__root.tsx",
        "src/integrations/**",
        ".output/**",
        "**/start.ts",
      ],
    },
  },
});
