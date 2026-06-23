import { expect, test } from "@playwright/test";

import { SEED_PRODUCTS } from "../setup/seed";
import { BackofficeProductPage } from "../pages/backoffice-product-page";

test("backoffice の商品管理でシード商品が一覧表示される", async ({ page }) => {
  await test.step("商品管理画面を開く", async () => {
    const backoffice = new BackofficeProductPage(page);
    await backoffice.goto();
    await expect(backoffice.heading).toBeVisible();
    await expect(backoffice.emptyMessage).toBeHidden();
  });

  await test.step("シード商品が SKU と商品名で並ぶ", async () => {
    const backoffice = new BackofficeProductPage(page);
    for (const seeded of SEED_PRODUCTS) {
      const row = backoffice.row(seeded.sku);
      await expect(row).toBeVisible();
      await expect(row).toContainText(seeded.name);
    }
  });
});
