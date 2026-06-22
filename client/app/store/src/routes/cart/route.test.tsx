import { afterEach, describe, expect, it } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";

import { type CartItem, readCart, writeCart } from "@/entities/cart";
import Cart from "./route";

function renderCart() {
  render(
    <MemoryRouter>
      <Cart />
    </MemoryRouter>,
  );
}

const seed: CartItem[] = [{ productId: 1, name: "りんご", priceCents: 12300, quantity: 2 }];

afterEach(() => localStorage.clear());

describe("Cart", () => {
  describe("正常系", () => {
    it("カートの明細と合計を描画する", async () => {
      writeCart(seed);
      renderCart();

      expect(await screen.findByText("りんご")).toBeDefined();
      expect(screen.getByText("¥246")).toBeDefined(); // 12300*2/100
    });

    it("増やすボタンで数量が増える", async () => {
      writeCart(seed);
      renderCart();

      const inc = await screen.findByRole("button", { name: "増やす" });
      fireEvent.click(inc);

      await waitFor(() => expect(readCart()[0].quantity).toBe(3));
    });

    it("削除でカートから消える", async () => {
      writeCart(seed);
      renderCart();

      const del = await screen.findByText("削除");
      fireEvent.click(del);

      await waitFor(() => expect(readCart()).toEqual([]));
    });
  });

  describe("準正常系", () => {
    it("空のとき空メッセージを描画する", async () => {
      renderCart();
      expect(await screen.findByText("カートは空です。")).toBeDefined();
    });
  });
});
