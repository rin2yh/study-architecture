import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

// useLoaderData をスタブ可能にするためのモック。
// vi.mock はファイル先頭に巻き上げられるため、参照する mock 関数も vi.hoisted で先に定義する。
const { useLoaderData } = vi.hoisted(() => ({ useLoaderData: vi.fn() }));

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (options: Record<string, unknown>) => ({
    ...options,
    useLoaderData,
  }),
}));

// createServerFn / loader は対象外。handler を素通しするスタブ。
vi.mock("@tanstack/react-start", () => ({
  createServerFn: () => ({
    handler: (fn: unknown) => fn,
  }),
}));

import { Route } from "./index";

const routeOptions = Route as unknown as {
  component: () => React.ReactElement;
  pendingComponent: () => React.ReactElement;
  errorComponent: (props: { error: Error }) => React.ReactElement;
};
const Home = routeOptions.component;
const Pending = routeOptions.pendingComponent;
const ErrorView = routeOptions.errorComponent;

afterEach(() => {
  cleanup();
  useLoaderData.mockReset();
});

describe("mypage Home", () => {
  it("注文履歴の行を描画する", () => {
    useLoaderData.mockReturnValue([
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

    render(<Home />);

    expect(screen.getByText("注文履歴")).toBeDefined();
    expect(screen.getByText("101")).toBeDefined();
    expect(screen.getByText("102")).toBeDefined();
    expect(screen.getByText("paid")).toBeDefined();
    expect(screen.getByText("shipped")).toBeDefined();
    expect(screen.getByText("¥300")).toBeDefined();
    expect(screen.getByText("¥500")).toBeDefined();
    expect(screen.queryByText("注文履歴がありません。")).toBeNull();
  });

  it("空のとき空メッセージを描画する", () => {
    useLoaderData.mockReturnValue([]);

    render(<Home />);

    expect(screen.getByText("注文履歴")).toBeDefined();
    expect(screen.getByText("注文履歴がありません。")).toBeDefined();
  });
});

describe("mypage route fallbacks", () => {
  it("pendingComponent はローディング表示を返す", () => {
    render(<Pending />);
    expect(screen.getByRole("status").textContent).toContain("読み込み中");
  });

  it("errorComponent はエラーメッセージを描画する", () => {
    render(<ErrorView error={new Error("order 取得失敗")} />);
    expect(screen.getByRole("alert").textContent).toContain("エラー");
    expect(screen.getByText("order 取得失敗")).toBeDefined();
  });
});
