import { execSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { seedForApp } from "../setup/seed";
import type { App } from "./apps";
import { appConfig } from "./apps";
import { waitForOk } from "./wait";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../../..");

function runScript(command: string): void {
  execSync(command, { cwd: repoRoot, stdio: "inherit" });
}

export async function bringUp(app: App): Promise<void> {
  runScript(`bash scripts/e2e-up.sh ${app} ${appConfig[app].profile}`);
  await seedForApp(app);
  // frontend は healthcheck を持たない (compose の --wait では待てない)。
  await waitForOk(appConfig[app].baseURL);
}

export function tearDown(app: App): void {
  runScript(`bash scripts/e2e-down.sh ${appConfig[app].profile}`);
}
