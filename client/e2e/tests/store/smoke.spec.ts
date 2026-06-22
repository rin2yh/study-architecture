import { expect, test } from "@playwright/test";

import { SEED_PRODUCTS } from "../setup/seed";

const seeded = SEED_PRODUCTS[0];

test.describe("store smoke", () => {
  test("商品一覧が SSR loader 経由で表示される", async ({ page }) => {
    await page.goto("/");

    await expect(page.getByRole("heading", { name: "商品一覧" })).toBeVisible();
    await expect(page.getByText(seeded.name, { exact: true })).toBeVisible();
  });

  test("カートに入れてカート画面で確認できる", async ({ page }) => {
    await page.goto("/");

    // ボタンは hydration 完了 (cart.ready) まで disabled。click の auto-wait で有効化を待つ。
    await page.getByRole("button", { name: "カートに入れる" }).first().click();

    await page.getByRole("link", { name: "カート" }).click();
    await expect(page).toHaveURL(/\/cart$/);

    await expect(page.getByRole("heading", { name: "カート" })).toBeVisible();
    await expect(page.getByText("カートは空です。")).toBeHidden();
    await expect(page.getByText("合計")).toBeVisible();
  });

  test("チェックアウト画面まで遷移しフォームが出る", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "カートに入れる" }).first().click();
    await page.getByRole("link", { name: "カート" }).click();

    await page.getByRole("link", { name: "チェックアウトへ" }).click();
    await expect(page).toHaveURL(/\/checkout$/);

    await expect(page.getByRole("heading", { name: "チェックアウト" })).toBeVisible();
    await expect(page.getByText("支払い方法")).toBeVisible();

    // 認証 (#6) は未導入。未ログインでの確定はログイン要求になることだけ固定し、action 経路を通す。
    await page.getByRole("button", { name: "注文を確定する" }).click();
    await expect(page.getByText("ログインが必要です。")).toBeVisible();
  });
});
