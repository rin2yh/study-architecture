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

const Home = (Route as unknown as { component: () => React.ReactElement }).component;

afterEach(() => {
  cleanup();
  useLoaderData.mockReset();
});

describe("backoffice Home", () => {
  it("商品管理の行を描画する", () => {
    useLoaderData.mockReturnValue([
      {
        id: 1,
        sku: "SKU-1",
        name: "りんご",
        priceCents: 12300,
        createdAt: "2026-01-01T00:00:00Z",
      },
      {
        id: 2,
        sku: "SKU-2",
        name: "みかん",
        priceCents: 45600,
        createdAt: "2026-01-02T00:00:00Z",
      },
    ]);

    render(<Home />);

    expect(screen.getByText("商品管理")).toBeDefined();
    expect(screen.getByText("りんご")).toBeDefined();
    expect(screen.getByText("みかん")).toBeDefined();
    expect(screen.getByText("SKU-1")).toBeDefined();
    expect(screen.getByText("SKU-2")).toBeDefined();
    expect(screen.getByText("¥123")).toBeDefined();
    expect(screen.getByText("¥456")).toBeDefined();
    expect(screen.queryByText("商品がありません。")).toBeNull();
  });

  it("空のとき空メッセージを描画する", () => {
    useLoaderData.mockReturnValue([]);

    render(<Home />);

    expect(screen.getByText("商品管理")).toBeDefined();
    expect(screen.getByText("商品がありません。")).toBeDefined();
  });
});
