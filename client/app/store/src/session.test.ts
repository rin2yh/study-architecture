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
  it("正常系 該当 Cookie のトークンを返す (前後の他 Cookie も無視)", () => {
    expect(readSessionToken(reqWithCookie(`foo=bar; ${SESSION_COOKIE}=tok123; baz=qux`))).toBe(
      "tok123",
    );
  });

  it("正常系 URL エンコードされた値をデコードする", () => {
    expect(readSessionToken(reqWithCookie(`${SESSION_COOKIE}=a%2Fb%3Dc`))).toBe("a/b=c");
  });

  it("準正常系 Cookie ヘッダ無しは null", () => {
    expect(readSessionToken(reqWithCookie(null))).toBeNull();
  });

  it("準正常系 該当 Cookie が無ければ null", () => {
    expect(readSessionToken(reqWithCookie("other=1"))).toBeNull();
  });
});

describe("currentMemberId", () => {
  afterEach(() => vi.clearAllMocks());

  it("正常系 有効なセッションは memberId を返す", async () => {
    vi.mocked(getSession).mockResolvedValue({
      data: { id: "tok123", memberId: 7, expiresAt: "2026-07-01T00:00:00Z" },
      status: 200,
      headers: new Headers(),
    } as Awaited<ReturnType<typeof getSession>>);

    expect(await currentMemberId(reqWithCookie(`${SESSION_COOKIE}=tok123`))).toBe(7);
    expect(vi.mocked(getSession).mock.calls[0][0]).toBe("tok123");
  });

  it("準正常系 トークン無しは getSession を呼ばず null", async () => {
    expect(await currentMemberId(reqWithCookie(null))).toBeNull();
    expect(vi.mocked(getSession)).not.toHaveBeenCalled();
  });

  it("異常系 getSession が throw したら null", async () => {
    vi.mocked(getSession).mockRejectedValue(new Error("404"));
    expect(await currentMemberId(reqWithCookie(`${SESSION_COOKIE}=tok123`))).toBeNull();
  });
});
