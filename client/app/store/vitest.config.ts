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
        "src/cart.ts",
        "src/use-cart.ts",
        "src/session.ts",
        "src/money.ts",
        "src/routes/home.tsx",
        "src/routes/cart.tsx",
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
