import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { createRoutesStub } from "react-router";

import Home, { ErrorBoundary, HydrateFallback, loader } from "./home";
import { currentMemberId } from "@/entities/session";
import { listOrders } from "api/order";

vi.mock("@/entities/session", () => ({ currentMemberId: vi.fn() }));
vi.mock("api/order", async (importActual) => {
  const actual = await importActual<typeof import("api/order")>();
  return { ...actual, listOrders: vi.fn() };
});

type Order = {
  id: number;
  memberId: number;
  status: string;
  totalCents: number;
  createdAt: string;
};

function renderHome(memberId: number, orders: Order[]) {
  const Comp = Home as unknown as (props: {
    loaderData: { memberId: number; orders: Order[] };
  }) => React.ReactElement;
  // Form が router context を要求するため stub でラップする。
  const Stub = createRoutesStub([
    { path: "/", Component: () => <Comp loaderData={{ memberId, orders }} /> },
  ]);
  render(<Stub initialEntries={["/"]} />);
}

function loaderArgs(request: Request) {
  return { request } as unknown as Parameters<typeof loader>[0];
}

describe("mypage Home loader", () => {
  afterEach(() => vi.clearAllMocks());

  describe("正常系", () => {
    it("ログイン済みなら X-Member-Id を付けて注文を返す", async () => {
      vi.mocked(currentMemberId).mockResolvedValue(7);
      vi.mocked(listOrders).mockResolvedValue({
        data: [
          {
            id: 1,
            memberId: 7,
            status: "paid",
            totalCents: 1000,
            createdAt: "2026-01-01T00:00:00Z",
          },
        ],
        status: 200,
        headers: new Headers(),
      } as Awaited<ReturnType<typeof listOrders>>);

      const result = await loader(loaderArgs(new Request("http://mypage.test/")));

      expect(vi.mocked(listOrders).mock.calls[0][0]).toEqual({ headers: { "X-Member-Id": "7" } });
      expect(result).toEqual({
        memberId: 7,
        orders: [
          {
            id: 1,
            memberId: 7,
            status: "paid",
            totalCents: 1000,
            createdAt: "2026-01-01T00:00:00Z",
          },
        ],
      });
    });
  });

  describe("準正常系", () => {
    it("未ログインなら /login へリダイレクトする", async () => {
      vi.mocked(currentMemberId).mockResolvedValue(null);

      const thrown = await loader(loaderArgs(new Request("http://mypage.test/"))).catch((e) => e);

      expect(thrown).toBeInstanceOf(Response);
      expect((thrown as Response).status).toBe(302);
      expect((thrown as Response).headers.get("Location")).toBe("/login");
      expect(vi.mocked(listOrders)).not.toHaveBeenCalled();
    });
  });
});

describe("mypage Home", () => {
  describe("正常系", () => {
    it("注文履歴の行と会員ID/ログアウトを描画する", () => {
      renderHome(7, [
        {
          id: 101,
          memberId: 7,
          status: "paid",
          totalCents: 30000,
          createdAt: "2026-01-01T00:00:00Z",
        },
        {
          id: 102,
          memberId: 9,
          status: "shipped",
          totalCents: 50000,
          createdAt: "2026-02-02T00:00:00Z",
        },
      ]);

      expect(screen.getByText("注文履歴")).toBeDefined();
      expect(screen.getByText("会員ID: 7")).toBeDefined();
      expect(screen.getByRole("button", { name: "ログアウト" })).toBeDefined();
      expect(screen.getByText("101")).toBeDefined();
      expect(screen.getByText("paid")).toBeDefined();
      expect(screen.getByText("¥300")).toBeDefined();
      expect(screen.queryByText("注文履歴がありません。")).toBeNull();
    });
  });

  describe("準正常系", () => {
    it("空のとき空メッセージを描画する", () => {
      renderHome(7, []);
      expect(screen.getByText("注文履歴")).toBeDefined();
      expect(screen.getByText("注文履歴がありません。")).toBeDefined();
    });
  });
});

describe("mypage route fallbacks", () => {
  describe("準正常系", () => {
    it("HydrateFallback はローディング表示を返す", () => {
      render(<HydrateFallback />);
      expect(screen.getByRole("status").textContent).toContain("読み込み中");
    });
  });

  describe("異常系", () => {
    it("ErrorBoundary はエラーメッセージを描画する", () => {
      const Boundary = ErrorBoundary as unknown as (props: {
        error: unknown;
      }) => React.ReactElement;
      render(<Boundary error={new Error("order 取得失敗")} />);
      expect(screen.getByRole("alert").textContent).toContain("エラー");
      expect(screen.getByText("order 取得失敗")).toBeDefined();
    });
  });
});
