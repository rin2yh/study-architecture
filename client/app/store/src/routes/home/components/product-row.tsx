import type { Product } from "api/product";

import { Badge } from "@/shared/ui/badge";
import { Card, CardContent } from "@/shared/ui/card";
import { AddToCartButton } from "@/features/add-to-cart";
import { useCart } from "@/entities/cart";
import { yen } from "@/shared/lib/money";

interface ProductRowProps {
  product: Product;
  cart: ReturnType<typeof useCart>;
}

export function ProductRow({ product, cart }: ProductRowProps) {
  return (
    <Card>
      <CardContent className="flex items-center justify-between">
        <div className="space-y-1">
          <p className="font-medium">{product.name}</p>
          <Badge variant="secondary">{product.sku}</Badge>
        </div>
        <div className="flex items-center gap-4">
          <span className="tabular-nums">{yen(product.priceCents)}</span>
          <AddToCartButton product={product} cart={cart} />
        </div>
      </CardContent>
    </Card>
  );
}
