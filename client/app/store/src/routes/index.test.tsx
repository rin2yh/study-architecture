import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

// useLoaderData をスタブ可能にするためのモック。
// createFileRoute は (path) => (options) => ({ ...options, useLoaderData }) を返し、
// index.tsx 内の `Route.useLoaderData()` をテストから差し替えられるようにする。
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

describe("store Home", () => {
  it("商品一覧の行を描画する", () => {
    useLoaderData.mockReturnValue([
      { id: 1, sku: "SKU-1", name: "りんご", priceCents: 12300, createdAt: "2026-01-01" },
      { id: 2, sku: "SKU-2", name: "みかん", priceCents: 45600, createdAt: "2026-01-02" },
    ]);

    render(<Home />);

    expect(screen.getByText("商品一覧")).toBeDefined();
    expect(screen.getByText("りんご")).toBeDefined();
    expect(screen.getByText("SKU-1")).toBeDefined();
    expect(screen.getByText("みかん")).toBeDefined();
    expect(screen.getByText("¥123")).toBeDefined();
    expect(screen.getByText("¥456")).toBeDefined();
    expect(screen.queryByText("商品がありません。")).toBeNull();
  });

  it("空のとき空メッセージを描画する", () => {
    useLoaderData.mockReturnValue([]);

    render(<Home />);

    expect(screen.getByText("商品一覧")).toBeDefined();
    expect(screen.getByText("商品がありません。")).toBeDefined();
  });
});

describe("store route fallbacks", () => {
  it("pendingComponent はローディング表示を返す", () => {
    render(<Pending />);
    expect(screen.getByRole("status").textContent).toContain("読み込み中");
  });

  it("errorComponent はエラーメッセージを描画する", () => {
    render(<ErrorView error={new Error("ネットワーク不通")} />);
    expect(screen.getByRole("alert").textContent).toContain("エラー");
    expect(screen.getByText("ネットワーク不通")).toBeDefined();
  });
});
