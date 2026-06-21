// カートはサーバに持たず、確定 (checkout) までクライアントの作業状態として localStorage に
// 置く ([[0008]] のカート観: スナップショットは確定時に order 側で取る)。配列変換は純粋関数に
// 切り出して読み書きの副作用と分離する。

export interface CartItem {
  productId: number;
  name: string;
  priceCents: number;
  quantity: number;
}

const STORAGE_KEY = "store.cart.v1";

export function readCart(): CartItem[] {
  if (typeof localStorage === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as CartItem[]) : [];
  } catch {
    return [];
  }
}

export function writeCart(items: CartItem[]): void {
  if (typeof localStorage === "undefined") return;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
}

export function addToCart(
  items: CartItem[],
  product: Omit<CartItem, "quantity">,
  quantity = 1,
): CartItem[] {
  const exists = items.some((i) => i.productId === product.productId);
  if (exists) {
    return items.map((i) =>
      i.productId === product.productId ? { ...i, quantity: i.quantity + quantity } : i,
    );
  }
  return [...items, { ...product, quantity }];
}

export function setQuantity(items: CartItem[], productId: number, quantity: number): CartItem[] {
  if (quantity <= 0) return removeFromCart(items, productId);
  return items.map((i) => (i.productId === productId ? { ...i, quantity } : i));
}

export function removeFromCart(items: CartItem[], productId: number): CartItem[] {
  return items.filter((i) => i.productId !== productId);
}

export function cartTotalCents(items: CartItem[]): number {
  return items.reduce((sum, i) => sum + i.priceCents * i.quantity, 0);
}

// checkout に送るのは productId と quantity のみ。商品名・単価は order が product を参照して
// 権威ある値でスナップショットするため、ここでの表示用の値は送らない ([[0008]])。
export function toCheckoutItems(items: CartItem[]): { productId: number; quantity: number }[] {
  return items.map((i) => ({ productId: i.productId, quantity: i.quantity }));
}
