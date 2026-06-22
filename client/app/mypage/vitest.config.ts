import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";
import viteReact from "@vitejs/plugin-react";

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
        "src/routes/home.tsx",
        "src/routes/login.tsx",
        "src/routes/logout.tsx",
        "src/entities/session/model/session.ts",
        "src/features/**/*.tsx",
        "src/pages/**/ui/*.tsx",
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
