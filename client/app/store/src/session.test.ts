import { afterEach, describe, expect, it, vi } from "vitest";

import { SESSION_COOKIE, currentMemberId, readSessionToken } from "./session";
import { getSession } from "api/member";

vi.mock("api/member", async (importActual) => {
  const actual = await importActual<typeof import("api/member")>();
  return { ...actual, getSession: vi.fn() };
});

function reqWithCookie(cookie: string | null): Request {
  const headers = new Headers();
  if (cookie !== null) headers.set("Cookie", cookie);
  return new Request("http://store.test/", { headers });
}

describe("readSessionToken", () => {
  describe("正常系", () => {
    it.each([
      [
        "該当 Cookie のトークンを返す (前後の他 Cookie も無視)",
        `foo=bar; ${SESSION_COOKIE}=tok123; baz=qux`,
        "tok123",
      ],
      ["URL エンコードされた値をデコードする", `${SESSION_COOKIE}=a%2Fb%3Dc`, "a/b=c"],
    ])("%s", (_name, cookie, expected) => {
      expect(readSessionToken(reqWithCookie(cookie))).toBe(expected);
    });
  });

  describe("準正常系", () => {
    it.each([
      ["Cookie ヘッダ無しは null", null],
      ["該当 Cookie が無ければ null", "other=1"],
    ])("%s", (_name, cookie) => {
      expect(readSessionToken(reqWithCookie(cookie))).toBeNull();
    });
  });
});

describe("currentMemberId", () => {
  afterEach(() => vi.clearAllMocks());

  describe("正常系", () => {
    it("有効なセッションは memberId を返す", async () => {
      vi.mocked(getSession).mockResolvedValue({
        data: { id: "tok123", memberId: 7, expiresAt: "2026-07-01T00:00:00Z" },
        status: 200,
        headers: new Headers(),
      } as Awaited<ReturnType<typeof getSession>>);

      expect(await currentMemberId(reqWithCookie(`${SESSION_COOKIE}=tok123`))).toBe(7);
      expect(vi.mocked(getSession).mock.calls[0][0]).toBe("tok123");
    });
  });

  describe("準正常系", () => {
    it("トークン無しは getSession を呼ばず null", async () => {
      expect(await currentMemberId(reqWithCookie(null))).toBeNull();
      expect(vi.mocked(getSession)).not.toHaveBeenCalled();
    });
  });

  describe("異常系", () => {
    it("getSession が throw したら null", async () => {
      vi.mocked(getSession).mockRejectedValue(new Error("404"));
      expect(await currentMemberId(reqWithCookie(`${SESSION_COOKIE}=tok123`))).toBeNull();
    });
  });
});
