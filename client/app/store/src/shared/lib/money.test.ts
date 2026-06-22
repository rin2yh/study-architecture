import { describe, expect, it } from "vitest";

import { yen } from "./money";

describe("yen", () => {
  describe("正常系", () => {
    it.each([
      [12300, "¥123"],
      [4560, "¥45.6"],
      [0, "¥0"],
      [100000, "¥1,000"],
    ])("yen(%i) = %s", (cents, expected) => {
      expect(yen(cents)).toBe(expected);
    });
  });
});
