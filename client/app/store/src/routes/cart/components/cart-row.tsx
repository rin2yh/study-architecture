import { Minus, Plus, Trash2 } from "lucide-react";

import { Button } from "ui/button";
import type { CartItem } from "@/entities/cart";
import { yen } from "@/shared/lib/money";

interface CartRowProps {
  item: CartItem;
  onSetQty: (productId: number, quantity: number) => void;
  onRemove: (productId: number) => void;
}

export function CartRow({ item, onSetQty, onRemove }: CartRowProps) {
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
