import type { Product } from "api/product";

import { Button } from "@/shared/ui/button";
import { useCart } from "@/entities/cart";

interface AddToCartButtonProps {
  product: Product;
  cart: ReturnType<typeof useCart>;
}

export function AddToCartButton({ product, cart }: AddToCartButtonProps) {
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
