import type { Order } from "api/order";
import type { Product } from "api/product";
import type { CartItem } from "@/entities/cart";

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

export const order: Order = {
  id: 1042,
  memberId: 1,
  status: "confirmed",
  totalCents: 580000,
  createdAt: "2026-06-01T00:00:00Z",
  items: [
    { productId: 1, productName: "コーヒー豆 200g", unitPriceCents: 180000, quantity: 2 },
    { productId: 2, productName: "ドリッパー", unitPriceCents: 220000, quantity: 1 },
  ],
};
