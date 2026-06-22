import { defineConfig, devices } from "@playwright/test";

// store とバックエンドは compose で起動済みの前提で叩く (ブラウザ側 route mock は SSR loader の
// サーバ fetch を差し替えられないため、実スタックに対して E2E する)。URL は CI/ローカルで
// 差し替えられるよう env 経由にする。
const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:5173";

export default defineConfig({
  testDir: "./tests",
  globalSetup: "./tests/setup/seed.ts",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL,
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
