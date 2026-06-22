import { useCallback, useEffect, useState } from "react";

import { addToCart, type CartItem, readCart, removeFromCart, setQuantity, writeCart } from "./cart";

// localStorage は SSR で読めないため、ready (マウント後) までカート依存 UI を出さず
// hydration 不一致を避ける。
export function useCart() {
  const [items, setItems] = useState<CartItem[]>([]);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    setItems(readCart());
    setReady(true);
  }, []);

  const apply = useCallback((fn: (prev: CartItem[]) => CartItem[]) => {
    setItems((prev) => {
      const next = fn(prev);
      writeCart(next);
      return next;
    });
  }, []);

  const add = useCallback(
    (product: Omit<CartItem, "quantity">) => apply((prev) => addToCart(prev, product)),
    [apply],
  );
  const setQty = useCallback(
    (productId: number, quantity: number) =>
      apply((prev) => setQuantity(prev, productId, quantity)),
    [apply],
  );
  const remove = useCallback(
    (productId: number) => apply((prev) => removeFromCart(prev, productId)),
    [apply],
  );
  const clear = useCallback(() => apply(() => []), [apply]);

  return { items, ready, add, setQty, remove, clear };
}
