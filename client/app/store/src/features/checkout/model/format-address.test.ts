import { describe, expect, it } from "vitest";

import { formatAddress } from "./format-address";

describe("formatAddress", () => {
  describe("正常系", () => {
    it("宛名・郵便番号・都道府県市区町村・番地を1行にまとめる", () => {
      expect(
        formatAddress({
          id: 5,
          memberId: 1,
          recipient: "山田太郎",
          postalCode: "1500001",
          prefecture: "東京都",
          city: "渋谷区",
          line1: "神宮前1-2-3",
          createdAt: "2026-01-01T00:00:00Z",
        }),
      ).toBe("山田太郎 / 〒1500001 東京都渋谷区 神宮前1-2-3");
    });
  });
});
