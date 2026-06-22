import { type Locator, type Page } from "@playwright/test";

export class CartPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly emptyMessage: Locator;
  readonly total: Locator;
  readonly checkoutLink: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", { name: "カート" });
    this.emptyMessage = page.getByText("カートは空です。");
    this.total = page.getByText("合計");
    this.checkoutLink = page.getByRole("link", { name: "チェックアウトへ" });
  }

  async openCheckout(): Promise<void> {
    await this.checkoutLink.click();
  }
}
