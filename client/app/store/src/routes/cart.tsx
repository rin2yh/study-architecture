import { Link } from "react-router";
import { Minus, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { cartTotalCents } from "../cart";
import { yen } from "../money";
import { useCart } from "../use-cart";

export default function Cart() {
  const cart = useCart();

  if (!cart.ready) {
    return (
      <div className="mx-auto max-w-2xl p-8 text-muted-foreground" role="status" aria-live="polite">
        読み込み中…
      </div>
    );
  }

  if (cart.items.length === 0) {
    return (
      <div className="mx-auto max-w-2xl p-8">
        <h1 className="text-3xl font-bold">カート</h1>
        <p className="mt-6 text-muted-foreground">カートは空です。</p>
        <Button asChild variant="link" className="mt-4 px-0">
          <Link to="/">商品一覧へ</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold">カート</h1>
      <Card className="mt-6">
        <CardContent className="space-y-3">
          {cart.items.map((i) => (
            <div key={i.productId} className="flex items-center justify-between">
              <div>
                <p className="font-medium">{i.name}</p>
                <p className="text-sm text-muted-foreground">{yen(i.priceCents)}</p>
              </div>
              <div className="flex items-center gap-3">
                <Button
                  variant="outline"
                  size="icon"
                  aria-label="減らす"
                  onClick={() => cart.setQty(i.productId, i.quantity - 1)}
                >
                  <Minus />
                </Button>
                <span className="w-6 text-center tabular-nums">{i.quantity}</span>
                <Button
                  variant="outline"
                  size="icon"
                  aria-label="増やす"
                  onClick={() => cart.setQty(i.productId, i.quantity + 1)}
                >
                  <Plus />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-destructive"
                  onClick={() => cart.remove(i.productId)}
                >
                  <Trash2 />
                  削除
                </Button>
              </div>
            </div>
          ))}
          <Separator />
          <div className="flex items-center justify-between">
            <span className="text-lg font-bold">合計</span>
            <span className="text-lg font-bold tabular-nums">
              {yen(cartTotalCents(cart.items))}
            </span>
          </div>
        </CardContent>
      </Card>
      <Button asChild className="mt-6">
        <Link to="/checkout">チェックアウトへ</Link>
      </Button>
    </div>
  );
}
