import { expect, test } from "@playwright/test";

import { SEED_PRODUCTS } from "../setup/seed";
import { SESSION_COOKIE, loginToken } from "../setup/auth";
import { CartPage } from "../pages/cart-page";
import { CheckoutPage } from "../pages/checkout-page";
import { HomePage } from "../pages/home-page";

const seeded = SEED_PRODUCTS[0];
const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:5173";

test("商品一覧からチェックアウトまでの購入フロー", async ({ page, context }) => {
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

  await test.step("ログイン済みで注文を確定する", async () => {
    // store にログイン UI が無いため (認証画面は mypage 側)。
    const token = await loginToken();
    await context.addCookies([{ name: SESSION_COOKIE, value: token, url: baseURL }]);

    await cart.openCheckout();
    await expect(page).toHaveURL(/\/checkout$/);
    await expect(checkout.heading).toBeVisible();
    await expect(checkout.paymentMethodLabel).toBeVisible();

    await checkout.submit();
    await expect(checkout.confirmedHeading).toBeVisible();
  });
});
