import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { productFetch } from "./mutator";

// 各サービスバレルを import し、re-export がロードされること(副作用なくモジュール解決できること)を確認する。
import * as productBarrel from "./product";
import * as orderBarrel from "./order";
import * as paymentBarrel from "./payment";
import * as memberBarrel from "./member";
import * as shippingBarrel from "./shipping";

describe("mutator productFetch", () => {
  beforeEach(() => {
    process.env.PRODUCT_API_URL = "https://product.example.test";
  });

  afterEach(() => {
    vi.restoreAllMocks();
    delete process.env.PRODUCT_API_URL;
  });

  it("(a) ok のとき data/status/headers を返し、baseURL を前置する", async () => {
    const headers = new Headers({ "content-type": "application/json" });
    const fetchMock = vi.fn(async () => ({
      ok: true,
      status: 200,
      statusText: "OK",
      body: {},
      headers,
      json: async () => [{ id: 1, sku: "SKU-1", name: "商品", priceCents: 1000, createdAt: "x" }],
    }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await productFetch<{ data: unknown; status: number; headers: Headers }>(
      "/products",
    );

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toBe("https://product.example.test/products");
    // content-type ヘッダが付与される
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect((init.headers as Record<string, string>)["content-type"]).toBe("application/json");
    expect(result.status).toBe(200);
    expect(result.headers).toBe(headers);
    expect(result.data).toEqual([
      { id: 1, sku: "SKU-1", name: "商品", priceCents: 1000, createdAt: "x" },
    ]);
  });

  it("(b) 非ok のとき throw する", async () => {
    const fetchMock = vi.fn(async () => ({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      body: {},
      headers: new Headers(),
      json: async () => ({ message: "boom" }),
    }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(productFetch("/products")).rejects.toThrow(
      "request to /products failed: 500 Internal Server Error",
    );
  });

  it("(c) 204 のとき data=undefined を返す", async () => {
    const fetchMock = vi.fn(async () => ({
      ok: true,
      status: 204,
      statusText: "No Content",
      body: null,
      headers: new Headers(),
      json: async () => {
        throw new Error("json は呼ばれてはならない");
      },
    }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await productFetch<{ data: unknown; status: number }>("/products");
    expect(result.data).toBeUndefined();
    expect(result.status).toBe(204);
  });

  it("env 未設定のとき baseURL は空文字になる", async () => {
    delete process.env.PRODUCT_API_URL;
    const fetchMock = vi.fn(async () => ({
      ok: true,
      status: 200,
      statusText: "OK",
      body: {},
      headers: new Headers(),
      json: async () => ({}),
    }));
    vi.stubGlobal("fetch", fetchMock);

    await productFetch("/health");
    expect(fetchMock.mock.calls[0][0]).toBe("/health");
  });
});

describe("service barrels", () => {
  it("各バレルが re-export をロードできる", () => {
    expect(productBarrel).toBeTypeOf("object");
    expect(orderBarrel).toBeTypeOf("object");
    expect(paymentBarrel).toBeTypeOf("object");
    expect(memberBarrel).toBeTypeOf("object");
    expect(shippingBarrel).toBeTypeOf("object");
    // 代表的な生成エクスポートが re-export されていること
    expect(productBarrel).toHaveProperty("listProducts");
    expect(orderBarrel).toHaveProperty("listOrders");
    expect(paymentBarrel).toHaveProperty("listPayments");
    expect(memberBarrel).toHaveProperty("listMembers");
    expect(shippingBarrel).toHaveProperty("listShipments");
  });
});
