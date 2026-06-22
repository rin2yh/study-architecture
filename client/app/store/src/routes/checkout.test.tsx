import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createRoutesStub } from "react-router";

import { checkout } from "api/order";
import { type CartItem, readCart, writeCart } from "../cart";
import { currentMemberId } from "../session";
import Checkout, { action } from "./checkout";

vi.mock("api/order", () => ({ checkout: vi.fn() }));
vi.mock("../session", () => ({ currentMemberId: vi.fn() }));

const order = {
  id: 7,
  memberId: 1,
  status: "confirmed",
  totalCents: 24600,
  createdAt: "2026-01-01T00:00:00Z",
  items: [{ productId: 1, productName: "りんご", unitPriceCents: 12300, quantity: 2 }],
};

const seed: CartItem[] = [{ productId: 1, name: "りんご", priceCents: 12300, quantity: 2 }];

function postRequest(fields: Record<string, string>) {
  const fd = new FormData();
  for (const [k, v] of Object.entries(fields)) fd.set(k, v);
  return new Request("http://test/checkout", { method: "POST", body: fd });
}

function callAction(fields: Record<string, string>) {
  return action({ request: postRequest(fields), params: {}, context: {} } as never);
}

function renderCheckout(actionResult?: unknown) {
  const Stub = createRoutesStub([
    { path: "/checkout", Component: Checkout as never, action: () => actionResult ?? null },
  ]);
  render(<Stub initialEntries={["/checkout"]} />);
}

afterEach(() => {
  localStorage.clear();
  vi.clearAllMocks();
});

describe("action", () => {
  describe("正常系", () => {
    it("カートと支払い方法を渡すと checkout を呼び注文を返す", async () => {
      vi.mocked(currentMemberId).mockResolvedValue(1);
      vi.mocked(checkout).mockResolvedValue({ data: order, status: 201 } as never);

      const result = await callAction({
        items: JSON.stringify([{ productId: 1, quantity: 2 }]),
        paymentMethod: "card",
      });

      expect(checkout).toHaveBeenCalledWith({
        memberId: 1,
        paymentMethod: "card",
        items: [{ productId: 1, quantity: 2 }],
      });
      expect(result).toEqual({ ok: true, order });
    });
  });

  describe("準正常系", () => {
    it("カートが空なら呼ばずにエラーを返す", async () => {
      const result = await callAction({ items: "[]", paymentMethod: "card" });
      expect(checkout).not.toHaveBeenCalled();
      expect(result).toEqual({ ok: false, error: "カートが空です。" });
    });

    it("支払い方法が無ければエラーを返す", async () => {
      const result = await callAction({ items: JSON.stringify([{ productId: 1, quantity: 1 }]) });
      expect(result).toEqual({ ok: false, error: "支払い方法を選択してください。" });
    });

    it("未ログインなら checkout を呼ばずエラーを返す", async () => {
      vi.mocked(currentMemberId).mockResolvedValue(null);
      const result = await callAction({
        items: JSON.stringify([{ productId: 1, quantity: 1 }]),
        paymentMethod: "card",
      });
      expect(checkout).not.toHaveBeenCalled();
      expect(result).toEqual({ ok: false, error: "ログインが必要です。" });
    });
  });

  describe("異常系", () => {
    it("checkout が失敗したらエラーメッセージを返す", async () => {
      vi.mocked(currentMemberId).mockResolvedValue(1);
      vi.mocked(checkout).mockRejectedValue(new Error("boom"));
      const result = await callAction({
        items: JSON.stringify([{ productId: 1, quantity: 1 }]),
        paymentMethod: "card",
      });
      expect(result).toEqual({ ok: false, error: "boom" });
    });
  });
});

describe("Checkout 画面", () => {
  describe("正常系", () => {
    it("カート明細と支払い方法フォームを描画する", async () => {
      writeCart(seed);
      renderCheckout();
      expect(await screen.findByText("チェックアウト")).toBeDefined();
      expect(screen.getByRole("button", { name: "注文を確定する" })).toBeDefined();
    });

    it("確定すると完了画面を出しカートを空にする", async () => {
      writeCart(seed);
      renderCheckout({ ok: true, order });

      fireEvent.click(await screen.findByRole("button", { name: "注文を確定する" }));

      expect(await screen.findByText("注文が確定しました")).toBeDefined();
      await waitFor(() => expect(readCart()).toEqual([]));
    });
  });

  describe("準正常系", () => {
    it("カートが空なら空メッセージを描画する", async () => {
      renderCheckout();
      expect(await screen.findByText("カートが空です。")).toBeDefined();
    });
  });

  describe("異常系", () => {
    it("action がエラーを返すとアラートを描画する", async () => {
      writeCart(seed);
      renderCheckout({ ok: false, error: "在庫切れ" });

      fireEvent.click(await screen.findByRole("button", { name: "注文を確定する" }));

      expect(await screen.findByRole("alert")).toHaveProperty("textContent", "在庫切れ");
    });
  });
});
