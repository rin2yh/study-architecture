import { defineConfig, devices } from "@playwright/test";

import { apps, baseURLs } from "./tests/stack/apps";

// webServer はグローバルで project 単位に持てないため、スタックの起動/停止は setup/teardown
// project に寄せる。--project=store なら Playwright が依存の store-setup だけ走らせるので、
// 起動するスタックも自然にその project に絞られる。
export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    trace: "on-first-retry",
  },
  projects: apps.flatMap((app) => [
    {
      name: `${app}-setup`,
      testMatch: new RegExp(`stack/${app}\\.setup\\.ts$`),
      teardown: `${app}-teardown`,
    },
    {
      name: `${app}-teardown`,
      testMatch: /stack\/teardown\.ts$/,
    },
    {
      name: app,
      testDir: `./tests/${app}`,
      dependencies: [`${app}-setup`],
      use: { ...devices["Desktop Chrome"], baseURL: baseURLs[app] },
    },
  ]),
});
