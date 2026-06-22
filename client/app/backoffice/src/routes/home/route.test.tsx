import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { createRoutesStub } from "react-router";

import Home, { ErrorBoundary, HydrateFallback } from "./route";

type Product = {
  id: number;
  sku: string;
  name: string;
  priceCents: number;
  createdAt: string;
};

function renderHome(products: Product[]) {
  const Stub = createRoutesStub([{ path: "/", Component: Home, loader: () => products }]);
  render(<Stub initialEntries={["/"]} />);
}

describe("backoffice Home", () => {
  it("商品管理の行を描画する", async () => {
    renderHome([
      { id: 1, sku: "SKU-1", name: "りんご", priceCents: 12300, createdAt: "2026-01-01T00:00:00Z" },
      { id: 2, sku: "SKU-2", name: "みかん", priceCents: 45600, createdAt: "2026-01-02T00:00:00Z" },
    ]);

    expect(await screen.findByText("商品管理")).toBeDefined();
    expect(screen.getByText("りんご")).toBeDefined();
    expect(screen.getByText("SKU-1")).toBeDefined();
    expect(screen.getByText("¥123")).toBeDefined();
    expect(screen.queryByText("商品がありません。")).toBeNull();
  });

  it("空のとき空メッセージを描画する", async () => {
    renderHome([]);
    expect(await screen.findByText("商品管理")).toBeDefined();
    expect(screen.getByText("商品がありません。")).toBeDefined();
  });
});

describe("backoffice route fallbacks", () => {
  it("HydrateFallback はローディング表示を返す", () => {
    render(<HydrateFallback />);
    expect(screen.getByRole("status").textContent).toContain("読み込み中");
  });

  it("ErrorBoundary はエラーメッセージを描画する", async () => {
    const Stub = createRoutesStub([
      {
        path: "/",
        Component: () => null,
        ErrorBoundary,
        loader: () => {
          throw new Error("product 取得失敗");
        },
      },
    ]);
    render(<Stub initialEntries={["/"]} />);
    expect((await screen.findByRole("alert")).textContent).toContain("エラー");
    expect(screen.getByText("product 取得失敗")).toBeDefined();
  });
});
