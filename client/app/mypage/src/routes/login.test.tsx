import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { createRoutesStub } from "react-router";

import Login, { action, loader } from "./login";
import { currentMemberId, SESSION_COOKIE } from "../session";
import { createSession } from "api/member";

vi.mock("../session", async (importActual) => {
  const actual = await importActual<typeof import("../session")>();
  return { ...actual, currentMemberId: vi.fn() };
});
vi.mock("api/member", async (importActual) => {
  const actual = await importActual<typeof import("api/member")>();
  return { ...actual, createSession: vi.fn() };
});

function loaderArgs(request: Request) {
  return { request } as unknown as Parameters<typeof loader>[0];
}
function actionArgs(request: Request) {
  return { request } as unknown as Parameters<typeof action>[0];
}
function postForm(fields: Record<string, string>): Request {
  return new Request("http://mypage.test/login", {
    method: "POST",
    body: new URLSearchParams(fields),
  });
}

describe("login loader", () => {
  afterEach(() => vi.clearAllMocks());

  it("準正常系 既ログインなら / へリダイレクト", async () => {
    vi.mocked(currentMemberId).mockResolvedValue(7);
    const thrown = await loader(loaderArgs(new Request("http://mypage.test/login"))).catch(
      (e) => e,
    );
    expect(thrown).toBeInstanceOf(Response);
    expect((thrown as Response).headers.get("Location")).toBe("/");
  });

  it("正常系 未ログインなら null を返しフォームを出す", async () => {
    vi.mocked(currentMemberId).mockResolvedValue(null);
    expect(await loader(loaderArgs(new Request("http://mypage.test/login")))).toBeNull();
  });
});

describe("login action", () => {
  afterEach(() => vi.clearAllMocks());

  it("正常系 認証成功で Set-Cookie 付きで / へリダイレクト", async () => {
    vi.mocked(createSession).mockResolvedValue({
      data: { id: "tok123", memberId: 7, expiresAt: "2026-07-01T00:00:00Z" },
      status: 201,
      headers: new Headers(),
    } as Awaited<ReturnType<typeof createSession>>);

    const res = await action(
      actionArgs(postForm({ email: "user@example.com", password: "password123" })),
    );

    expect(res).toBeInstanceOf(Response);
    const response = res as Response;
    expect(response.headers.get("Location")).toBe("/");
    expect(response.headers.get("Set-Cookie")).toContain(`${SESSION_COOKIE}=tok123`);
  });

  it("準正常系 認証失敗はエラー文言を返す (リダイレクトしない)", async () => {
    vi.mocked(createSession).mockRejectedValue(new Error("401"));

    const res = await action(
      actionArgs(postForm({ email: "user@example.com", password: "wrong" })),
    );

    expect(res).toEqual({ error: "メールアドレスまたはパスワードが違います" });
  });
});

describe("Login component", () => {
  function renderLogin(actionData?: { error: string }) {
    const Comp = Login as unknown as (props: {
      actionData?: { error: string };
    }) => React.ReactElement;
    // Form が router context を要求するため stub でラップする。
    const Stub = createRoutesStub([
      { path: "/login", Component: () => <Comp actionData={actionData} /> },
    ]);
    render(<Stub initialEntries={["/login"]} />);
  }

  it("正常系 エラー文言を描画する", () => {
    renderLogin({ error: "認証に失敗" });
    expect(screen.getByRole("alert").textContent).toContain("認証に失敗");
    expect(screen.getByRole("button", { name: "ログイン" })).toBeDefined();
  });

  it("準正常系 actionData 無しでもフォームを描画する", () => {
    renderLogin();
    expect(screen.getByRole("button", { name: "ログイン" })).toBeDefined();
    expect(screen.queryByRole("alert")).toBeNull();
  });
});
