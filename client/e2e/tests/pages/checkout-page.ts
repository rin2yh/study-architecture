import { type Locator, type Page } from "@playwright/test";

export class CheckoutPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly paymentMethodLabel: Locator;
  readonly submitButton: Locator;
  readonly confirmedHeading: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole("heading", { name: "チェックアウト" });
    this.paymentMethodLabel = page.getByText("支払い方法");
    this.submitButton = page.getByRole("button", { name: "注文を確定する" });
    this.confirmedHeading = page.getByRole("heading", { name: "注文が確定しました" });
  }

  async submit(): Promise<void> {
    await this.submitButton.click();
  }
}
