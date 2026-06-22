import { Link } from "react-router";
import { ShoppingCart } from "lucide-react";

import type { Product } from "api/product";

import { Button } from "@/shared/ui/button";
import { useCart } from "@/entities/cart";
import { ProductRow } from "./product-row";

export function HomePage({ loaderData }: { loaderData: Product[] }) {
  const products = loaderData;
  const cart = useCart();
  return (
    <div className="mx-auto max-w-2xl p-8">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">商品一覧</h1>
        <Button asChild variant="outline" size="sm">
          <Link to="/cart">
            <ShoppingCart />
            カート
          </Link>
        </Button>
      </div>
      <div className="mt-6 space-y-3">
        {products.map((p) => (
          <ProductRow key={p.id} product={p} cart={cart} />
        ))}
      </div>
      {products.length === 0 && <p className="mt-6 text-muted-foreground">商品がありません。</p>}
    </div>
  );
}
