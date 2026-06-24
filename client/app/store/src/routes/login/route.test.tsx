import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { createRoutesStub } from "react-router";

import Login, { action, loader } from "./route";
import { redirectIfAuthenticated, SESSION_COOKIE } from "@/features/auth/model/session";
import { createSession } from "api/member";
import { redirect } from "react-router";

vi.mock("@/features/auth/model/session", async (importActual) => {
  const actual = await importActual<typeof import("@/features/auth/model/session")>();
  return { ...actual, redirectIfAuthenticated: vi.fn() };
});
vi.mock("api/member", async (importActual) => {
  const actual = await importActual<typeof import("api/member")>();
  return { ...actual, createSession: vi.fn() };
});

function loaderArgs(request: Request): Parameters<typeof loader>[0] {
  return { request, url: new URL(request.url), params: {}, pattern: "/login", context: {} };
}
function actionArgs(request: Request): Parameters<typeof action>[0] {
  return { request, url: new URL(request.url), params: {}, pattern: "/login", context: {} };
}
function postForm(fields: Record<string, string>): Request {
  return new Request("http://store.test/login", {
    method: "POST",
    body: new URLSearchParams(fields),
  });
}

describe("login loader", () => {
  afterEach(() => vi.clearAllMocks());

  describe("正常系", () => {
    it("未ログインなら null を返しフォームを出す", async () => {
      vi.mocked(redirectIfAuthenticated).mockResolvedValue(undefined);
      expect(await loader(loaderArgs(new Request("http://store.test/login")))).toBeNull();
    });
  });

  describe("準正常系", () => {
    it("既ログインなら / へリダイレクト", async () => {
      vi.mocked(redirectIfAuthenticated).mockRejectedValue(redirect("/"));
      const thrown: unknown = await loader(
        loaderArgs(new Request("http://store.test/login")),
      ).catch((e: unknown) => e);
      expect(thrown).toBeInstanceOf(Response);
      if (!(thrown instanceof Response)) throw thrown;
      expect(thrown.headers.get("Location")).toBe("/");
    });
  });
});

describe("login action", () => {
  afterEach(() => vi.clearAllMocks());

  describe("正常系", () => {
    it("認証成功で Set-Cookie 付きで / へリダイレクト", async () => {
      vi.mocked(createSession).mockResolvedValue({
        data: { id: "tok123", memberId: 7, expiresAt: "2026-07-01T00:00:00Z" },
        status: 201,
        headers: new Headers(),
      });

      const res = await action(
        actionArgs(postForm({ email: "user@example.com", password: "password123" })),
      );

      expect(res).toBeInstanceOf(Response);
      if (!(res instanceof Response)) throw res;
      expect(res.headers.get("Location")).toBe("/");
      expect(res.headers.get("Set-Cookie")).toContain(`${SESSION_COOKIE}=tok123`);
    });
  });

  describe("準正常系", () => {
    it("認証失敗はエラー文言を返す (リダイレクトしない)", async () => {
      vi.mocked(createSession).mockRejectedValue(new Error("401"));

      const res = await action(
        actionArgs(postForm({ email: "user@example.com", password: "wrong" })),
      );

      expect(res).toEqual({ error: "メールアドレスまたはパスワードが違います" });
    });
  });
});

describe("Login component", () => {
  function renderLogin(actionData?: { error: string }) {
    const Stub = createRoutesStub([{ id: "login", path: "/login", Component: Login }]);
    render(
      <Stub
        initialEntries={["/login"]}
        hydrationData={{ actionData: actionData ? { login: actionData } : null }}
      />,
    );
  }

  describe("正常系", () => {
    it("エラー文言を描画する", () => {
      renderLogin({ error: "認証に失敗" });
      expect(screen.getByRole("alert").textContent).toContain("認証に失敗");
      expect(screen.getByRole("button", { name: "ログイン" })).toBeDefined();
    });
  });

  describe("準正常系", () => {
    it("actionData 無しでもフォームを描画する", () => {
      renderLogin();
      expect(screen.getByRole("button", { name: "ログイン" })).toBeDefined();
      expect(screen.queryByRole("alert")).toBeNull();
    });
  });
});
