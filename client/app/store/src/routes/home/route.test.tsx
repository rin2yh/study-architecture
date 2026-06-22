import { afterEach, describe, expect, it } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createRoutesStub } from "react-router";

import { readCart } from "@/entities/cart";
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

afterEach(() => localStorage.clear());

describe("Home", () => {
  describe("正常系", () => {
    it("商品一覧の行を描画する", async () => {
      renderHome([
        { id: 1, sku: "SKU-1", name: "りんご", priceCents: 12300, createdAt: "2026-01-01" },
        { id: 2, sku: "SKU-2", name: "みかん", priceCents: 45600, createdAt: "2026-01-02" },
      ]);

      expect(await screen.findByText("商品一覧")).toBeDefined();
      expect(screen.getByText("りんご")).toBeDefined();
      expect(screen.getByText("SKU-1")).toBeDefined();
      expect(screen.getByText("¥123")).toBeDefined();
      expect(screen.getByText("¥456")).toBeDefined();
      expect(screen.queryByText("商品がありません。")).toBeNull();
    });

    it("カートに入れるとカートへ保存される", async () => {
      renderHome([
        { id: 1, sku: "SKU-1", name: "りんご", priceCents: 12300, createdAt: "2026-01-01" },
      ]);

      const button = await screen.findByRole<HTMLButtonElement>("button", {
        name: "カートに入れる",
      });
      await waitFor(() => expect(button.disabled).toBe(false));
      fireEvent.click(button);

      expect(readCart()).toEqual([
        { productId: 1, name: "りんご", priceCents: 12300, quantity: 1 },
      ]);
    });
  });

  describe("準正常系", () => {
    it("空のとき空メッセージを描画する", async () => {
      renderHome([]);
      expect(await screen.findByText("商品一覧")).toBeDefined();
      expect(screen.getByText("商品がありません。")).toBeDefined();
    });
  });
});

describe("route fallbacks", () => {
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
            throw new Error("ネットワーク不通");
          },
        },
      ]);
      render(<Stub initialEntries={["/"]} />);
      expect((await screen.findByRole("alert")).textContent).toContain("エラー");
      expect(screen.getByText("ネットワーク不通")).toBeDefined();
    });
  });
});
