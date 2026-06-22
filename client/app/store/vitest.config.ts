import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";
import viteReact from "@vitejs/plugin-react";

// React Router の dev plugin は build/SSR 用なので vitest では読まない。
export default defineConfig({
  plugins: [viteReact()],
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  test: {
    environment: "jsdom",
    globals: true,
    include: ["src/**/*.test.{ts,tsx}"],
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary", "lcov"],
      reportsDirectory: "./coverage",
      include: [
        "src/entities/cart/model/cart.ts",
        "src/entities/cart/model/use-cart.ts",
        "src/entities/session/model/session.ts",
        "src/shared/lib/money.ts",
        "src/features/**/*.{ts,tsx}",
        "src/pages/**/ui/*.tsx",
        "src/routes/checkout.tsx",
      ],
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
