import { defineConfig } from "vitest/config";
import viteReact from "@vitejs/plugin-react";

// テスト専用設定。React Router の dev plugin は build/SSR 用なので vitest では読まず、
// JSX 変換に必要な react プラグインのみ。
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
      include: ["src/routes/home.tsx"],
      exclude: ["**/*.config.*", "src/root.tsx", "src/routes.ts", ".react-router/**", "build/**"],
      thresholds: {
        lines: 60,
        statements: 60,
        functions: 60,
        branches: 60,
      },
    },
  },
});
