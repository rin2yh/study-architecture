import type { CartItem } from "@/entities/cart";
import type { CheckoutItemInput } from "./parse-items";

// ADR-[[202606190900]]
export function toCheckoutItems(items: CartItem[]): CheckoutItemInput[] {
  return items.map((i) => ({ productId: i.productId, quantity: i.quantity }));
}
