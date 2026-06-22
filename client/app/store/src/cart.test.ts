import { afterEach, describe, expect, it } from "vitest";

import {
  addToCart,
  type CartItem,
  cartTotalCents,
  readCart,
  removeFromCart,
  setQuantity,
  toCheckoutItems,
  writeCart,
} from "./cart";

const apple: Omit<CartItem, "quantity"> = { productId: 1, name: "りんご", priceCents: 12300 };
const orange: Omit<CartItem, "quantity"> = { productId: 2, name: "みかん", priceCents: 4560 };

afterEach(() => localStorage.clear());

describe("cart transforms", () => {
  describe("正常系", () => {
    it("addToCart は新規商品を quantity=1 で追加する", () => {
      expect(addToCart([], apple)).toEqual([{ ...apple, quantity: 1 }]);
    });

    it("addToCart は既存商品の数量を加算する", () => {
      const items = [{ ...apple, quantity: 2 }];
      expect(addToCart(items, apple)).toEqual([{ ...apple, quantity: 3 }]);
    });

    it("setQuantity は対象の数量を置き換える", () => {
      const items = [{ ...apple, quantity: 1 }];
      expect(setQuantity(items, 1, 5)).toEqual([{ ...apple, quantity: 5 }]);
    });

    it("removeFromCart は対象を取り除く", () => {
      const items = [
        { ...apple, quantity: 1 },
        { ...orange, quantity: 1 },
      ];
      expect(removeFromCart(items, 1)).toEqual([{ ...orange, quantity: 1 }]);
    });

    it("cartTotalCents は単価×数量の合計を返す", () => {
      const items = [
        { ...apple, quantity: 2 },
        { ...orange, quantity: 1 },
      ];
      expect(cartTotalCents(items)).toBe(12300 * 2 + 4560);
    });

    it("toCheckoutItems は productId と quantity のみに絞る", () => {
      const items = [{ ...apple, quantity: 3 }];
      expect(toCheckoutItems(items)).toEqual([{ productId: 1, quantity: 3 }]);
    });

    it("writeCart→readCart はラウンドトリップする", () => {
      const items = [{ ...apple, quantity: 2 }];
      writeCart(items);
      expect(readCart()).toEqual(items);
    });
  });

  describe("準正常系", () => {
    it("setQuantity に 0 以下を渡すと削除になる", () => {
      const items = [{ ...apple, quantity: 1 }];
      expect(setQuantity(items, 1, 0)).toEqual([]);
    });

    it("readCart は未保存なら空配列を返す", () => {
      expect(readCart()).toEqual([]);
    });
  });

  describe("異常系", () => {
    it("readCart は壊れた JSON を空配列にフォールバックする", () => {
      localStorage.setItem("store.cart.v1", "{not json");
      expect(readCart()).toEqual([]);
    });

    it("readCart は配列でない JSON を空配列にする", () => {
      localStorage.setItem("store.cart.v1", '{"a":1}');
      expect(readCart()).toEqual([]);
    });
  });
});
