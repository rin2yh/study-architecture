import type { Product } from "api/product";

import { Button } from "@/shared/ui/button";
import { useCart } from "@/entities/cart";

export function AddToCartButton({
  product,
  cart,
}: {
  product: Product;
  cart: ReturnType<typeof useCart>;
}) {
  return (
    <Button
      disabled={!cart.ready}
      onClick={() =>
        cart.add({ productId: product.id, name: product.name, priceCents: product.priceCents })
      }
    >
      カートに入れる
    </Button>
  );
}
