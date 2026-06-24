import { afterEach, describe, expect, it } from "vitest";

import {
  addToCart,
  type CartItem,
  cartTotalCents,
  readCart,
  removeFromCart,
  setQuantity,
  writeCart,
} from "./cart";

const apple: Omit<CartItem, "quantity"> = { productId: 1, name: "りんご", priceCents: 12300 };
const orange: Omit<CartItem, "quantity"> = { productId: 2, name: "みかん", priceCents: 4560 };

afterEach(() => localStorage.clear());

describe("addToCart", () => {
  describe("正常系", () => {
    it("新規商品を quantity=1 で追加する", () => {
      expect(addToCart([], apple)).toEqual([{ ...apple, quantity: 1 }]);
    });

    it("既存商品の数量を加算する", () => {
      expect(addToCart([{ ...apple, quantity: 2 }], apple)).toEqual([{ ...apple, quantity: 3 }]);
    });
  });
});

describe("setQuantity", () => {
  describe("正常系", () => {
    it("対象の数量を置き換える", () => {
      expect(setQuantity([{ ...apple, quantity: 1 }], 1, 5)).toEqual([{ ...apple, quantity: 5 }]);
    });
  });

  describe("準正常系", () => {
    it("0 以下を渡すと削除になる", () => {
      expect(setQuantity([{ ...apple, quantity: 1 }], 1, 0)).toEqual([]);
    });
  });
});

describe("removeFromCart", () => {
  describe("正常系", () => {
    it("対象を取り除く", () => {
      const items = [
        { ...apple, quantity: 1 },
        { ...orange, quantity: 1 },
      ];
      expect(removeFromCart(items, 1)).toEqual([{ ...orange, quantity: 1 }]);
    });
  });
});

describe("cartTotalCents", () => {
  describe("正常系", () => {
    it("単価×数量の合計を返す", () => {
      const items = [
        { ...apple, quantity: 2 },
        { ...orange, quantity: 1 },
      ];
      expect(cartTotalCents(items)).toBe(12300 * 2 + 4560);
    });
  });
});

describe("readCart", () => {
  describe("正常系", () => {
    it("writeCart で保存した内容を読み戻す", () => {
      const items = [{ ...apple, quantity: 2 }];
      writeCart(items);
      expect(readCart()).toEqual(items);
    });
  });

  describe("準正常系", () => {
    it("未保存なら空配列を返す", () => {
      expect(readCart()).toEqual([]);
    });
  });

  describe("異常系", () => {
    it.each([
      ["壊れた JSON", "{not json"],
      ["配列でない JSON", '{"a":1}'],
    ])("%sを空配列にフォールバックする", (_name, stored) => {
      localStorage.setItem("store.cart.v1", stored);
      expect(readCart()).toEqual([]);
    });
  });
});
