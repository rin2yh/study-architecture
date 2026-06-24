import { describe, expect, it } from "vitest";

import type { CartItem } from "@/entities/cart";
import { parseItems, toCheckoutItems } from "./checkout-items";

const apple: CartItem = { productId: 1, name: "りんご", priceCents: 12300, quantity: 3 };

describe("toCheckoutItems", () => {
  describe("正常系", () => {
    it("productId と quantity のみに絞る", () => {
      expect(toCheckoutItems([apple])).toEqual([{ productId: 1, quantity: 3 }]);
    });
  });
});

describe("parseItems", () => {
  describe("正常系", () => {
    it("JSON 配列をそのまま読む", () => {
      expect(parseItems(JSON.stringify([{ productId: 1, quantity: 2 }]))).toEqual([
        { productId: 1, quantity: 2 },
      ]);
    });
  });

  describe("準正常系", () => {
    it.each([
      ["null", null],
      ["壊れた JSON", "{not json"],
      ["配列でない JSON", '{"a":1}'],
    ])("%sを空配列にフォールバックする", (_name, raw) => {
      expect(parseItems(raw)).toEqual([]);
    });
  });
});
