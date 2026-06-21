import { describe, expect, it } from "vitest";

import { DEV_MEMBER_ID, getCurrentMemberId } from "./session";

describe("正常系 session", () => {
  it("getCurrentMemberId は暫定の開発用会員 ID を返す", () => {
    expect(getCurrentMemberId()).toBe(DEV_MEMBER_ID);
  });
});
