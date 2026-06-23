import { test as setup } from "@playwright/test";

import { appFromProjectName } from "./apps";
import { bringUp } from "./compose";

setup("スタックを起動してシードする", async () => {
  setup.setTimeout(600_000);
  await bringUp(appFromProjectName(setup.info().project.name));
});
