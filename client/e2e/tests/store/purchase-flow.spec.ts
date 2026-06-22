import { expect, test } from "@playwright/test";

import { SEED_PRODUCTS } from "../setup/seed";
import { MEMBER } from "../setup/auth";
import { CartPage } from "../pages/cart-page";
import { CheckoutPage } from "../pages/checkout-page";
import { HomePage } from "../pages/home-page";
import { LoginPage } from "../pages/login-page";

const seeded = SEED_PRODUCTS[0];

test("ログインから商品購入までのフロー", async ({ page }) => {
  const login = new LoginPage(page);
  const home = new HomePage(page);
  const cart = new CartPage(page);
  const checkout = new CheckoutPage(page);

  await test.step("/login からログインして商品一覧へ", async () => {
    await login.goto();
    await login.login(MEMBER.email, MEMBER.password);

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

  await test.step("注文を確定する", async () => {
    await cart.openCheckout();
    await expect(page).toHaveURL(/\/checkout$/);
    await expect(checkout.heading).toBeVisible();
    await expect(checkout.paymentMethodLabel).toBeVisible();

    await checkout.submit();
    await expect(checkout.confirmedHeading).toBeVisible();
  });
});
