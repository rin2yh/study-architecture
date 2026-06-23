import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { createRoutesStub } from "react-router";

import Orders, { ErrorBoundary, HydrateFallback, loader } from "./route";
import { requireMemberId } from "@/shared/lib/session";
import { listOrders } from "api/order";
import { redirect } from "react-router";

vi.mock("@/shared/lib/session", () => ({ requireMemberId: vi.fn() }));
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
  const Stub = createRoutesStub([
    { path: "/", Component: Orders, loader: () => ({ memberId, orders }) },
  ]);
  render(<Stub initialEntries={["/"]} />);
}

function loaderArgs(request: Request): Parameters<typeof loader>[0] {
  return { request, url: new URL(request.url), params: {}, pattern: "/", context: {} };
}

describe("Orders loader", () => {
  afterEach(() => vi.clearAllMocks());

  describe("正常系", () => {
    it("ログイン済みなら X-Member-Id を付けて注文を返す", async () => {
      vi.mocked(requireMemberId).mockResolvedValue(7);
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
      });

      const result = await loader(loaderArgs(new Request("http://store.test/")));

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
      vi.mocked(requireMemberId).mockRejectedValue(redirect("/login"));

      const thrown: unknown = await loader(loaderArgs(new Request("http://store.test/"))).catch(
        (e: unknown) => e,
      );

      expect(thrown).toBeInstanceOf(Response);
      if (!(thrown instanceof Response)) throw thrown;
      expect(thrown.status).toBe(302);
      expect(thrown.headers.get("Location")).toBe("/login");
      expect(vi.mocked(listOrders)).not.toHaveBeenCalled();
    });
  });
});

describe("Orders", () => {
  describe("正常系", () => {
    it("注文履歴の行と会員ID/ログアウトを描画する", async () => {
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

      expect(await screen.findByText("注文履歴")).toBeDefined();
      expect(screen.getByText("会員ID: 7")).toBeDefined();
      expect(screen.getByRole("button", { name: "ログアウト" })).toBeDefined();
      expect(screen.getByText("101")).toBeDefined();
      expect(screen.getByText("paid")).toBeDefined();
      expect(screen.getByText("¥300")).toBeDefined();
      expect(screen.queryByText("注文履歴がありません。")).toBeNull();
    });
  });

  describe("準正常系", () => {
    it("空のとき空メッセージを描画する", async () => {
      renderHome(7, []);
      expect(await screen.findByText("注文履歴")).toBeDefined();
      expect(screen.getByText("注文履歴がありません。")).toBeDefined();
    });
  });
});

describe("orders route fallbacks", () => {
  describe("準正常系", () => {
    it("HydrateFallback はローディング表示を返す", () => {
      render(<HydrateFallback />);
      expect(screen.getByRole("status").textContent).toContain("読み込み中");
    });
  });

  describe("異常系", () => {
    it("ErrorBoundary はエラーメッセージを描画する", async () => {
      const Stub = createRoutesStub([
        {
          path: "/",
          Component: () => null,
          ErrorBoundary,
          loader: () => {
            throw new Error("order 取得失敗");
          },
        },
      ]);
      render(<Stub initialEntries={["/"]} />);
      expect((await screen.findByRole("alert")).textContent).toContain("エラー");
      expect(screen.getByText("order 取得失敗")).toBeDefined();
    });
  });
});
