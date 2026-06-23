import { afterEach, describe, expect, it, vi } from "vitest";

import { listMyOrders } from "./order-api";
import { listOrders } from "api/order";

vi.mock("api/order", async (importActual) => {
  const actual = await importActual<typeof import("api/order")>();
  return { ...actual, listOrders: vi.fn() };
});

describe("listMyOrders", () => {
  afterEach(() => vi.clearAllMocks());

  describe("正常系", () => {
    it("検証済み memberId から X-Member-Id を付けて listOrders を呼ぶ", async () => {
      vi.mocked(listOrders).mockResolvedValue({ data: [], status: 200, headers: new Headers() });

      await listMyOrders(7);

      expect(vi.mocked(listOrders).mock.calls[0][0]).toEqual({ headers: { "X-Member-Id": "7" } });
    });
  });
});
