import { afterEach, describe, expect, it, vi } from "vitest";

import { action, loader } from "./route";
import { SESSION_COOKIE } from "@/entities/session";
import { deleteSession } from "api/member";

vi.mock("api/member", async (importActual) => {
  const actual = await importActual<typeof import("api/member")>();
  return { ...actual, deleteSession: vi.fn() };
});

function actionArgs(request: Request): Parameters<typeof action>[0] {
  return { request, url: new URL(request.url), params: {}, pattern: "/logout", context: {} };
}
function reqWithCookie(cookie: string | null): Request {
  const headers = new Headers();
  if (cookie !== null) headers.set("Cookie", cookie);
  return new Request("http://mypage.test/logout", { method: "POST", headers });
}

describe("logout action", () => {
  afterEach(() => vi.clearAllMocks());

  describe("正常系", () => {
    it("トークンがあれば破棄し Cookie を消して /login へ", async () => {
      vi.mocked(deleteSession).mockResolvedValue({
        data: undefined,
        status: 204,
        headers: new Headers(),
      });

      const res = await action(actionArgs(reqWithCookie(`${SESSION_COOKIE}=tok123`)));

      expect(vi.mocked(deleteSession).mock.calls[0][0]).toBe("tok123");
      expect(res.headers.get("Location")).toBe("/login");
      expect(res.headers.get("Set-Cookie")).toContain("Max-Age=0");
    });
  });

  describe("準正常系", () => {
    it("トークン無しでも /login へ (破棄は呼ばない)", async () => {
      const res = await action(actionArgs(reqWithCookie(null)));
      expect(vi.mocked(deleteSession)).not.toHaveBeenCalled();
      expect(res.headers.get("Location")).toBe("/login");
    });
  });

  describe("異常系", () => {
    it("破棄が失敗しても Cookie を消して /login へ", async () => {
      vi.mocked(deleteSession).mockRejectedValue(new Error("500"));
      const res = await action(actionArgs(reqWithCookie(`${SESSION_COOKIE}=tok123`)));
      expect(res.headers.get("Location")).toBe("/login");
      expect(res.headers.get("Set-Cookie")).toContain("Max-Age=0");
    });
  });
});

describe("logout loader", () => {
  describe("準正常系", () => {
    it("GET は /login へリダイレクト", async () => {
      const thrown: unknown = await loader().catch((e: unknown) => e);
      expect(thrown).toBeInstanceOf(Response);
      if (!(thrown instanceof Response)) throw thrown;
      expect(thrown.headers.get("Location")).toBe("/login");
    });
  });
});
