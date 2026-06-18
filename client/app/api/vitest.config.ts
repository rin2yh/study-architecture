import { defineConfig } from "vitest/config";

// api の手書き部分(mutator + サービスバレル)を検証する最小設定。
// orval 生成物は分母から除外する。
export default defineConfig({
  test: {
    environment: "node",
    globals: true,
    include: ["src/**/*.test.ts"],
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary", "lcov"],
      reportsDirectory: "./coverage",
      // 手書き: mutator + サービスバレル
      include: [
        "src/mutator.ts",
        "src/product.ts",
        "src/order.ts",
        "src/payment.ts",
        "src/member.ts",
        "src/shipping.ts",
      ],
      // 生成物・設定を除外
      exclude: [
        "src/product/**",
        "src/order/**",
        "src/payment/**",
        "src/member/**",
        "src/shipping/**",
        "**/*.gen.*",
        "**/*.config.*",
      ],
      thresholds: {
        lines: 60,
        statements: 60,
        functions: 60,
        branches: 60,
      },
    },
  },
});
