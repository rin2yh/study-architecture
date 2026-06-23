import { test as teardown } from "@playwright/test";

import { appFromProjectName } from "./apps";
import { tearDown } from "./compose";

teardown("スタックを停止する", async () => {
  teardown.setTimeout(120_000);
  tearDown(appFromProjectName(teardown.info().project.name));
});
