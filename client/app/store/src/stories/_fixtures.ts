import type { Order } from "api/order";
import type { Product } from "api/product";
import { cartTotalCents, type CartItem } from "@/entities/cart";

export const products: Product[] = [
  {
    id: 1,
    sku: "COF-200",
    name: "コーヒー豆 200g",
    priceCents: 180000,
    createdAt: "2026-06-01T00:00:00Z",
  },
  {
    id: 2,
    sku: "DRP-01",
    name: "ドリッパー",
    priceCents: 220000,
    createdAt: "2026-06-01T00:00:00Z",
  },
  {
    id: 3,
    sku: "FLT-100",
    name: "ペーパーフィルター 100枚",
    priceCents: 48000,
    createdAt: "2026-06-01T00:00:00Z",
  },
];

export const cartItems: CartItem[] = [
  { productId: 1, name: "コーヒー豆 200g", priceCents: 180000, quantity: 2 },
  { productId: 2, name: "ドリッパー", priceCents: 220000, quantity: 1 },
];

// cartItems と二重に持つと齟齬が出るため。
export const order: Order = {
  id: 1042,
  memberId: 1,
  status: "confirmed",
  createdAt: "2026-06-01T00:00:00Z",
  totalCents: cartTotalCents(cartItems),
  items: cartItems.map((i) => ({
    productId: i.productId,
    productName: i.name,
    unitPriceCents: i.priceCents,
    quantity: i.quantity,
  })),
};
