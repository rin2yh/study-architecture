import { expect, test } from "@playwright/test";

import { SEED_PRODUCTS } from "../setup/seed";
import { CartPage } from "../pages/cart-page";
import { CheckoutPage } from "../pages/checkout-page";
import { HomePage } from "../pages/home-page";

const seeded = SEED_PRODUCTS[0];

test("商品一覧からチェックアウトまでの購入フロー", async ({ page }) => {
  const home = new HomePage(page);
  const cart = new CartPage(page);
  const checkout = new CheckoutPage(page);

  await test.step("商品一覧が SSR loader 経由で表示される", async () => {
    await home.goto();
    await expect(home.heading).toBeVisible();
    await expect(home.product(seeded.name)).toBeVisible();
  });

  await test.step("カートに入れてカート画面で確認する", async () => {
    await home.addFirstToCart();
    await home.openCart();
    await expect(page).toHaveURL(/\/cart$/);
    await expect(cart.heading).toBeVisible();
    await expect(cart.emptyMessage).toBeHidden();
    await expect(cart.total).toBeVisible();
  });

  await test.step("チェックアウトへ進みフォームが出る", async () => {
    await cart.openCheckout();
    await expect(page).toHaveURL(/\/checkout$/);
    await expect(checkout.heading).toBeVisible();
    await expect(checkout.paymentMethodLabel).toBeVisible();
  });

  await test.step("未ログインの確定はログイン要求になる", async () => {
    // 認証 (#6) は未導入。action 経路を通し、ログイン要求になることだけ固定する。
    await checkout.submit();
    await expect(checkout.error("ログインが必要です。")).toBeVisible();
  });
});
