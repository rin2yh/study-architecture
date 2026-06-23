import { describe, expect, it } from "vitest";

import type { CartItem } from "@/entities/cart";
import { toCheckoutItems } from "./to-checkout-items";

const apple: CartItem = { productId: 1, name: "りんご", priceCents: 12300, quantity: 3 };

describe("toCheckoutItems", () => {
  describe("正常系", () => {
    it("productId と quantity のみに絞る", () => {
      expect(toCheckoutItems([apple])).toEqual([{ productId: 1, quantity: 3 }]);
    });
  });
});
