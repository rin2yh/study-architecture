import { defineConfig, devices } from "@playwright/test";

const appNames = ["store", "backoffice"] as const;
type App = (typeof appNames)[number];

const baseURLs: Record<App, string> = {
  store: process.env.E2E_BASE_URL ?? "http://localhost:5173",
  backoffice: process.env.E2E_BACKOFFICE_BASE_URL ?? "http://localhost:5175",
};

const selected = appNames.filter(
  (app) => !process.env.E2E_PROJECT || process.env.E2E_PROJECT === app,
);

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
  webServer: selected.map((app) => ({
    command: `bash scripts/e2e-up.sh ${app}`,
    cwd: "../..",
    url: baseURLs[app],
    reuseExistingServer: !process.env.CI,
    timeout: 600_000,
  })),
  projects: selected.map((app) => ({
    name: app,
    testDir: `./tests/${app}`,
    use: { ...devices["Desktop Chrome"], baseURL: baseURLs[app] },
  })),
});
