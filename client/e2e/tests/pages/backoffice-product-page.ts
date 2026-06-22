import { type Locator, type Page } from "@playwright/test";

// baseURL は store (社外スタック) を指すため、社内スタックの backoffice へは絶対 URL で遷移する。
const BACKOFFICE_BASE_URL = process.env.E2E_BACKOFFICE_BASE_URL ?? "http://localhost:5175";

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
    await this.page.goto(BACKOFFICE_BASE_URL);
  }

  // SKU は商品ごとに一意。
  row(sku: string): Locator {
    return this.page.getByRole("row").filter({ hasText: sku });
  }
}
