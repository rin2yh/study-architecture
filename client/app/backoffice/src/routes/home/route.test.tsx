import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import Home, { ErrorBoundary, HydrateFallback } from "./route";

type Product = {
  id: number;
  sku: string;
  name: string;
  priceCents: number;
  createdAt: string;
};

function renderHome(products: Product[]) {
  const Comp = Home as unknown as (props: { loaderData: Product[] }) => React.ReactElement;
  render(<Comp loaderData={products} />);
}

describe("backoffice Home", () => {
  it("商品管理の行を描画する", () => {
    renderHome([
      { id: 1, sku: "SKU-1", name: "りんご", priceCents: 12300, createdAt: "2026-01-01T00:00:00Z" },
      { id: 2, sku: "SKU-2", name: "みかん", priceCents: 45600, createdAt: "2026-01-02T00:00:00Z" },
    ]);

    expect(screen.getByText("商品管理")).toBeDefined();
    expect(screen.getByText("りんご")).toBeDefined();
    expect(screen.getByText("SKU-1")).toBeDefined();
    expect(screen.getByText("¥123")).toBeDefined();
    expect(screen.queryByText("商品がありません。")).toBeNull();
  });

  it("空のとき空メッセージを描画する", () => {
    renderHome([]);
    expect(screen.getByText("商品管理")).toBeDefined();
    expect(screen.getByText("商品がありません。")).toBeDefined();
  });
});

describe("backoffice route fallbacks", () => {
  it("HydrateFallback はローディング表示を返す", () => {
    render(<HydrateFallback />);
    expect(screen.getByRole("status").textContent).toContain("読み込み中");
  });

  it("ErrorBoundary はエラーメッセージを描画する", () => {
    const Boundary = ErrorBoundary as unknown as (props: { error: unknown }) => React.ReactElement;
    render(<Boundary error={new Error("product 取得失敗")} />);
    expect(screen.getByRole("alert").textContent).toContain("エラー");
    expect(screen.getByText("product 取得失敗")).toBeDefined();
  });
});
