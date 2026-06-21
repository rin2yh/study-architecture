import { Link } from "react-router";
import { ShoppingCart } from "lucide-react";

import { listProducts, ListProductsResponse } from "api/product";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { yen } from "../money";
import { useCart } from "../use-cart";
import type { Route } from "./+types/home";

export async function loader() {
  const { data } = await listProducts();
  return ListProductsResponse.parse(data);
}

export default function Home({ loaderData }: Route.ComponentProps) {
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
          <Card key={p.id}>
            <CardContent className="flex items-center justify-between">
              <div className="space-y-1">
                <p className="font-medium">{p.name}</p>
                <Badge variant="secondary">{p.sku}</Badge>
              </div>
              <div className="flex items-center gap-4">
                <span className="tabular-nums">{yen(p.priceCents)}</span>
                <Button
                  disabled={!cart.ready}
                  onClick={() =>
                    cart.add({ productId: p.id, name: p.name, priceCents: p.priceCents })
                  }
                >
                  カートに入れる
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
      {products.length === 0 && <p className="mt-6 text-muted-foreground">商品がありません。</p>}
    </div>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  const message = error instanceof Error ? error.message : "unknown error";
  return (
    <div className="mx-auto max-w-2xl p-8">
      <Alert variant="destructive">
        <AlertTitle>エラーが発生しました</AlertTitle>
        <AlertDescription>
          <p>商品一覧の取得に失敗しました。</p>
          <pre className="mt-2 overflow-x-auto text-xs">{message}</pre>
        </AlertDescription>
      </Alert>
    </div>
  );
}

export function HydrateFallback() {
  return (
    <div className="mx-auto max-w-2xl p-8 text-muted-foreground" role="status" aria-live="polite">
      読み込み中…
    </div>
  );
}
