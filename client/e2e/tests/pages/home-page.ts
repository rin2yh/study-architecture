import { type Locator, type Page } from "@playwright/test";

export class HomePage {
  readonly page: Page;
  readonly heading: Locator;
  readonly cartLink: Locator;
  private readonly addToCartButtons: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", { name: "商品一覧" });
    this.cartLink = page.getByRole("link", { name: "カート" });
    this.addToCartButtons = page.getByRole("button", { name: "カートに入れる" });
  }

  async goto(): Promise<void> {
    await this.page.goto("/");
  }

  product(name: string): Locator {
    return this.page.getByText(name, { exact: true });
  }

  // ボタンは hydration 完了 (cart.ready) まで disabled。click の auto-wait で有効化を待つ。
  async addFirstToCart(): Promise<void> {
    await this.addToCartButtons.first().click();
  }

  async openCart(): Promise<void> {
    await this.cartLink.click();
  }
}
