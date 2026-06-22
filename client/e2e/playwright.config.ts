import { defineConfig, devices } from "@playwright/test";

const storeBaseURL = process.env.E2E_BASE_URL ?? "http://localhost:5173";
const backofficeBaseURL = process.env.E2E_BACKOFFICE_BASE_URL ?? "http://localhost:5175";

// store (社外) と backoffice (社内) は別量子で毎回両方を動かすわけではないため、E2E_PROJECT で
// 起動するスタックを片方に絞れるようにする (PR #43 review)。未指定なら両方を順に動かす。
const target = process.env.E2E_PROJECT;
const runStore = !target || target === "store";
const runBackoffice = !target || target === "backoffice";

function webServerFor(app: "store" | "backoffice", url: string) {
  return {
    command: `bash scripts/e2e-up.sh ${app}`,
    cwd: "../..",
    url,
    reuseExistingServer: !process.env.CI,
    timeout: 600_000,
  };
}

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
  webServer: [
    ...(runStore ? [webServerFor("store", storeBaseURL)] : []),
    ...(runBackoffice ? [webServerFor("backoffice", backofficeBaseURL)] : []),
  ],
  projects: [
    {
      name: "store",
      testDir: "./tests/store",
      use: { ...devices["Desktop Chrome"], baseURL: storeBaseURL },
    },
    {
      name: "backoffice",
      testDir: "./tests/backoffice",
      use: { ...devices["Desktop Chrome"], baseURL: backofficeBaseURL },
    },
  ],
});
