import { Link } from "react-router";

import { Button } from "ui/button";
import { Card, CardContent } from "ui/card";
import { Separator } from "ui/separator";
import { type CartItem, cartTotalCents, useCart } from "@/entities/cart";
import { yen } from "@/shared/lib/money";
import { CartRow } from "./cart-row";

type Cart = ReturnType<typeof useCart>;

interface CartListProps {
  items: CartItem[];
  onSetQty: Cart["setQty"];
  onRemove: Cart["remove"];
}

export function CartList({ items, onSetQty, onRemove }: CartListProps) {
  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold">カート</h1>
      <Card className="mt-6">
        <CardContent className="space-y-3">
          {items.map((i) => (
            <CartRow key={i.productId} item={i} onSetQty={onSetQty} onRemove={onRemove} />
          ))}
          <Separator />
          <div className="flex items-center justify-between">
            <span className="text-lg font-bold">合計</span>
            <span className="text-lg font-bold tabular-nums">{yen(cartTotalCents(items))}</span>
          </div>
        </CardContent>
      </Card>
      <Button asChild className="mt-6">
        <Link to="/checkout">チェックアウトへ</Link>
      </Button>
    </div>
  );
}
