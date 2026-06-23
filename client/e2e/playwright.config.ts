import { defineConfig, devices } from "@playwright/test";

// スタックの起動は Playwright の外 (mise タスク / CI の scripts/e2e-up.sh) で行う。webServer を
// project 単位に持てないため、ここでは tests と seed (globalSetup) だけを扱う。
const baseURLs = {
  store: process.env.E2E_BASE_URL ?? "http://localhost:5173",
  backoffice: process.env.E2E_BACKOFFICE_BASE_URL ?? "http://localhost:5175",
};

export default defineConfig({
  testDir: "./tests",
  globalSetup: "./tests/setup/seed.ts",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "store",
      testDir: "./tests/store",
      use: { ...devices["Desktop Chrome"], baseURL: baseURLs.store },
    },
    {
      name: "backoffice",
      testDir: "./tests/backoffice",
      use: { ...devices["Desktop Chrome"], baseURL: baseURLs.backoffice },
    },
  ],
});
