import { afterEach, describe, expect, it, vi } from "vitest";

import {
  SESSION_COOKIE,
  clearSessionCookie,
  currentMemberId,
  readSessionToken,
  redirectIfAuthenticated,
  requireMemberId,
  sessionCookie,
} from "./session";
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

describe("sessionCookie / clearSessionCookie", () => {
  describe("正常系", () => {
    it("HttpOnly/SameSite/Path/Max-Age を含む", () => {
      const c = sessionCookie("tok123");
      expect(c).toContain(`${SESSION_COOKIE}=tok123`);
      expect(c).toContain("HttpOnly");
      expect(c).toContain("SameSite=Lax");
      expect(c).toContain("Path=/");
      expect(c).toMatch(/Max-Age=\d+/);
    });

    it("clear は Max-Age=0 で失効させる", () => {
      expect(clearSessionCookie()).toContain("Max-Age=0");
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
      });

      expect(await currentMemberId(reqWithCookie(`${SESSION_COOKIE}=tok123`))).toBe(7);
      expect(vi.mocked(getSession).mock.calls[0][0]).toBe("tok123");
    });
  });

  describe("準正常系", () => {
    it("トークン無しは getSession を呼ばず null", async () => {
      expect(await currentMemberId(reqWithCookie(null))).toBeNull();
      expect(vi.mocked(getSession)).not.toHaveBeenCalled();
    });

    it("セッション検証に失敗 (4xx) したら null", async () => {
      vi.mocked(getSession).mockRejectedValue(new Error("404"));
      expect(await currentMemberId(reqWithCookie(`${SESSION_COOKIE}=tok123`))).toBeNull();
    });
  });
});

function okSession(memberId: number) {
  vi.mocked(getSession).mockResolvedValue({
    data: { id: "tok123", memberId, expiresAt: "2026-07-01T00:00:00Z" },
    status: 200,
    headers: new Headers(),
  });
}

describe("requireMemberId", () => {
  afterEach(() => vi.clearAllMocks());

  describe("正常系", () => {
    it("有効なセッションは memberId を返す", async () => {
      okSession(7);
      expect(await requireMemberId(reqWithCookie(`${SESSION_COOKIE}=tok123`))).toBe(7);
    });
  });

  describe("準正常系", () => {
    it("未ログインは /login へ throw redirect", async () => {
      const thrown: unknown = await requireMemberId(reqWithCookie(null)).catch((e: unknown) => e);
      expect(thrown).toBeInstanceOf(Response);
      if (!(thrown instanceof Response)) throw thrown;
      expect(thrown.status).toBe(302);
      expect(thrown.headers.get("Location")).toBe("/login");
    });
  });
});

describe("redirectIfAuthenticated", () => {
  afterEach(() => vi.clearAllMocks());

  describe("正常系", () => {
    it("未ログインは throw せず通す", async () => {
      await expect(redirectIfAuthenticated(reqWithCookie(null))).resolves.toBeUndefined();
    });
  });

  describe("準正常系", () => {
    it("既ログインは指定先へ throw redirect", async () => {
      okSession(7);
      const thrown: unknown = await redirectIfAuthenticated(
        reqWithCookie(`${SESSION_COOKIE}=tok123`),
      ).catch((e: unknown) => e);
      expect(thrown).toBeInstanceOf(Response);
      if (!(thrown instanceof Response)) throw thrown;
      expect(thrown.headers.get("Location")).toBe("/");
    });
  });
});
