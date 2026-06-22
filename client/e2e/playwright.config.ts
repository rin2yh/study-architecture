import { defineConfig, devices } from "@playwright/test";

const appNames = ["store", "backoffice"] as const;
type App = (typeof appNames)[number];

const baseURLs: Record<App, string> = {
  store: process.env.E2E_BASE_URL ?? "http://localhost:5173",
  backoffice: process.env.E2E_BACKOFFICE_BASE_URL ?? "http://localhost:5175",
};

// webServer は --project では絞られないため、CLI の --project を自前で読んで起動スタックも
// 同じ project に合わせる。未指定なら両方。
function selectProjects(): readonly App[] {
  const picked = new Set<string>();
  process.argv.forEach((arg, i) => {
    if (arg === "--project") picked.add(process.argv[i + 1] ?? "");
    else if (arg.startsWith("--project=")) picked.add(arg.slice("--project=".length));
  });
  const valid = appNames.filter((app) => picked.has(app));
  return valid.length > 0 ? valid : appNames;
}
const selected = selectProjects();

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
