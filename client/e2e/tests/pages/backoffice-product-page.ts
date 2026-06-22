import { type Locator, type Page } from "@playwright/test";

export class BackofficeProductPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly emptyMessage: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", { name: "商品管理" });
    this.emptyMessage = page.getByText("商品がありません。");
  }

  async goto(): Promise<void> {
    await this.page.goto("/");
  }

  // SKU は商品ごとに一意。
  row(sku: string): Locator {
    return this.page.getByRole("row").filter({ hasText: sku });
  }
}
