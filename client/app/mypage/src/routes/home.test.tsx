import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import Home, { ErrorBoundary, HydrateFallback } from "./home";

type Order = {
  id: number;
  memberId: number;
  status: string;
  totalCents: number;
  createdAt: string;
};

function renderHome(orders: Order[]) {
  const Comp = Home as unknown as (props: { loaderData: Order[] }) => React.ReactElement;
  render(<Comp loaderData={orders} />);
}

describe("mypage Home", () => {
  it("注文履歴の行を描画する", () => {
    renderHome([
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
    expect(screen.getByText("101")).toBeDefined();
    expect(screen.getByText("paid")).toBeDefined();
    expect(screen.getByText("¥300")).toBeDefined();
    expect(screen.queryByText("注文履歴がありません。")).toBeNull();
  });

  it("空のとき空メッセージを描画する", () => {
    renderHome([]);
    expect(screen.getByText("注文履歴")).toBeDefined();
    expect(screen.getByText("注文履歴がありません。")).toBeDefined();
  });
});

describe("mypage route fallbacks", () => {
  it("HydrateFallback はローディング表示を返す", () => {
    render(<HydrateFallback />);
    expect(screen.getByRole("status").textContent).toContain("読み込み中");
  });

  it("ErrorBoundary はエラーメッセージを描画する", () => {
    const Boundary = ErrorBoundary as unknown as (props: { error: unknown }) => React.ReactElement;
    render(<Boundary error={new Error("order 取得失敗")} />);
    expect(screen.getByRole("alert").textContent).toContain("エラー");
    expect(screen.getByText("order 取得失敗")).toBeDefined();
  });
});
