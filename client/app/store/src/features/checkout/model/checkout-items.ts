import type { CartItem } from "@/entities/cart";

export interface CheckoutItemInput {
  productId: number;
  quantity: number;
}

export function parseItems(raw: FormDataEntryValue | null): CheckoutItemInput[] {
  try {
    const parsed: unknown = JSON.parse(String(raw ?? "[]"));
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

// ADR-[[202606190900]]
export function toCheckoutItems(items: CartItem[]): CheckoutItemInput[] {
  return items.map((i) => ({ productId: i.productId, quantity: i.quantity }));
}
