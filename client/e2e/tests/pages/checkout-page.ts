import { type Locator, type Page } from "@playwright/test";

export class CheckoutPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly paymentMethodLabel: Locator;
  readonly submitButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", { name: "チェックアウト" });
    this.paymentMethodLabel = page.getByText("支払い方法");
    this.submitButton = page.getByRole("button", { name: "注文を確定する" });
  }

  error(message: string): Locator {
    return this.page.getByText(message);
  }

  async submit(): Promise<void> {
    await this.submitButton.click();
  }
}
