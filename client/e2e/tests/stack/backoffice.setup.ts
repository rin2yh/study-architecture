import { test as setup } from "@playwright/test";

import { bringUp } from "./compose";

setup("backoffice スタックを起動してシードする", async () => {
  setup.setTimeout(600_000);
  await bringUp("backoffice");
});
