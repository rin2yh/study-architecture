import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createRoutesStub } from "react-router";

import { checkout } from "api/order";
import { type CartItem, readCart, writeCart } from "@/entities/cart";
import { currentMemberId } from "@/features/auth/model/session";
import Checkout, { action } from "./route";

vi.mock("api/order", () => ({ checkout: vi.fn() }));
vi.mock("@/features/auth/model/session", () => ({ currentMemberId: vi.fn() }));

const order = {
  id: 7,
  memberId: 1,
  status: "confirmed",
  totalCents: 24600,
  createdAt: "2026-01-01T00:00:00Z",
  items: [{ productId: 1, productName: "りんご", unitPriceCents: 12300, quantity: 2 }],
};

const seed: CartItem[] = [{ productId: 1, name: "りんご", priceCents: 12300, quantity: 2 }];

const addresses = [
  {
    id: 5,
    memberId: 1,
    recipient: "山田太郎",
    postalCode: "1500001",
    prefecture: "東京都",
    city: "渋谷区",
    line1: "神宮前1-2-3",
    createdAt: "2026-01-01T00:00:00Z",
  },
];

function postRequest(fields: Record<string, string>) {
  const fd = new FormData();
  for (const [k, v] of Object.entries(fields)) fd.set(k, v);
  return new Request("http://test/checkout", { method: "POST", body: fd });
}

function callAction(fields: Record<string, string>) {
  const request = postRequest(fields);
  return action({
    request,
    url: new URL(request.url),
    params: {},
    pattern: "/checkout",
    context: {},
  });
}

function renderCheckout(actionResult?: unknown, loaderAddresses = addresses) {
  const Stub = createRoutesStub([
    {
      path: "/checkout",
      Component: Checkout,
      loader: () => ({ addresses: loaderAddresses }),
      action: () => actionResult ?? null,
    },
  ]);
  render(<Stub initialEntries={["/checkout"]} />);
}

afterEach(() => {
  localStorage.clear();
  vi.clearAllMocks();
});

describe("action", () => {
  describe("正常系", () => {
    it("カート・配送先・支払い方法を渡すと checkout を呼び注文を返す", async () => {
      vi.mocked(currentMemberId).mockResolvedValue(1);
      vi.mocked(checkout).mockResolvedValue({ data: order, status: 201, headers: new Headers() });

      const result = await callAction({
        items: JSON.stringify([{ productId: 1, quantity: 2 }]),
        shippingAddressId: "5",
        paymentMethod: "card",
      });

      expect(checkout).toHaveBeenCalledWith({
        memberId: 1,
        shippingAddressId: 5,
        paymentMethod: "card",
        items: [{ productId: 1, quantity: 2 }],
      });
      expect(result).toEqual({ ok: true, order });
    });
  });

  describe("準正常系", () => {
    const oneItem = JSON.stringify([{ productId: 1, quantity: 1 }]);
    it.each([
      [
        "カートが空",
        { items: "[]", paymentMethod: "card", shippingAddressId: "5" },
        "カートが空です。",
      ],
      [
        "支払い方法が未指定",
        { items: oneItem, shippingAddressId: "5" },
        "支払い方法を選択してください。",
      ],
      ["配送先が未選択", { items: oneItem, paymentMethod: "card" }, "配送先を選択してください。"],
      [
        "未ログイン",
        { items: oneItem, paymentMethod: "card", shippingAddressId: "5" },
        "ログインが必要です。",
      ],
    ])("%s なら checkout を呼ばずエラーを返す", async (_name, fields, error) => {
      vi.mocked(currentMemberId).mockResolvedValue(null);
      const result = await callAction(fields);
      expect(checkout).not.toHaveBeenCalled();
      expect(result).toEqual({ ok: false, error });
    });
  });

  describe("異常系", () => {
    it("checkout が失敗したらエラーメッセージを返す", async () => {
      vi.mocked(currentMemberId).mockResolvedValue(1);
      vi.mocked(checkout).mockRejectedValue(new Error("boom"));
      const result = await callAction({
        items: JSON.stringify([{ productId: 1, quantity: 1 }]),
        shippingAddressId: "5",
        paymentMethod: "card",
      });
      expect(result).toEqual({ ok: false, error: "boom" });
    });
  });
});

describe("Checkout 画面", () => {
  describe("正常系", () => {
    it("カート明細と配送先・支払い方法フォームを描画する", async () => {
      writeCart(seed);
      renderCheckout();
      expect(await screen.findByText("チェックアウト")).toBeDefined();
      expect(screen.getByText("配送先")).toBeDefined();
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

    it("配送先が未登録なら登録を促し確定ボタンを出さない", async () => {
      writeCart(seed);
      renderCheckout(undefined, []);
      expect(
        await screen.findByText("配送先が登録されていません。住所を登録してから注文してください。"),
      ).toBeDefined();
      expect(screen.queryByRole("button", { name: "注文を確定する" })).toBeNull();
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
