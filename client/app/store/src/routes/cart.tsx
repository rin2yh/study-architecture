import { Link } from "react-router";
import { Minus, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { PageLoading } from "@/components/page-loading";
import { type CartItem, cartTotalCents } from "../cart";
import { yen } from "../money";
import { useCart } from "../use-cart";

type Cart = ReturnType<typeof useCart>;

function EmptyCart() {
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

function CartRow({
  item,
  onSetQty,
  onRemove,
}: {
  item: CartItem;
  onSetQty: (productId: number, quantity: number) => void;
  onRemove: (productId: number) => void;
}) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <p className="font-medium">{item.name}</p>
        <p className="text-sm text-muted-foreground">{yen(item.priceCents)}</p>
      </div>
      <div className="flex items-center gap-3">
        <Button
          variant="outline"
          size="icon"
          aria-label="減らす"
          onClick={() => onSetQty(item.productId, item.quantity - 1)}
        >
          <Minus />
        </Button>
        <span className="w-6 text-center tabular-nums">{item.quantity}</span>
        <Button
          variant="outline"
          size="icon"
          aria-label="増やす"
          onClick={() => onSetQty(item.productId, item.quantity + 1)}
        >
          <Plus />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="text-destructive"
          onClick={() => onRemove(item.productId)}
        >
          <Trash2 />
          削除
        </Button>
      </div>
    </div>
  );
}

function CartList({
  items,
  onSetQty,
  onRemove,
}: {
  items: CartItem[];
  onSetQty: Cart["setQty"];
  onRemove: Cart["remove"];
}) {
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

export default function Cart() {
  const cart = useCart();

  if (!cart.ready) return <PageLoading />;
  if (cart.items.length === 0) return <EmptyCart />;
  return <CartList items={cart.items} onSetQty={cart.setQty} onRemove={cart.remove} />;
}
