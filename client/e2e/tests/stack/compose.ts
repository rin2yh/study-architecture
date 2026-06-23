import { execSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { seedForApp } from "../setup/seed";
import type { App } from "./apps";
import { baseURLs } from "./apps";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../../..");

function runScript(command: string): void {
  execSync(command, { cwd: repoRoot, stdio: "inherit" });
}

async function waitForUrl(url: string, timeoutMs = 120_000): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  let lastError: unknown;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(url);
      if (res.ok) return;
      lastError = new Error(`GET ${url} returned ${res.status}`);
    } catch (e) {
      lastError = e;
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`frontend not ready at ${url}: ${String(lastError)}`);
}

// frontend は healthcheck を持たないため、compose up 後に baseURL の到達を待ってから tests に渡す。
export async function bringUp(app: App): Promise<void> {
  runScript(`bash scripts/e2e-up.sh ${app}`);
  await seedForApp(app);
  await waitForUrl(baseURLs[app]);
}

export function tearDown(): void {
  runScript("bash scripts/e2e-down.sh");
}
