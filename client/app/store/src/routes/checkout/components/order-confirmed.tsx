import { Link } from "react-router";
import { CheckCircle2 } from "lucide-react";

import type { Order } from "api/order";

import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { yen } from "@/shared/lib/money";

interface OrderConfirmedProps {
  order: Order;
}

export function OrderConfirmed({ order }: OrderConfirmedProps) {
  return (
    <div className="mx-auto max-w-2xl p-8">
      <div className="flex items-center gap-2 text-green-600">
        <CheckCircle2 />
        <h1 className="text-3xl font-bold text-foreground">注文が確定しました</h1>
      </div>
      <Card className="mt-6">
        <CardHeader>
          <CardTitle>
            注文番号 <span className="font-mono">#{order.id}</span> / 合計{" "}
            <span className="tabular-nums">{yen(order.totalCents)}</span>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {(order.items ?? []).map((it) => (
            <div key={it.productId} className="flex justify-between">
              <span>
                {it.productName} × {it.quantity}
              </span>
              <span className="tabular-nums">{yen(it.unitPriceCents * it.quantity)}</span>
            </div>
          ))}
        </CardContent>
      </Card>
      <Button asChild variant="link" className="mt-6 px-0">
        <Link to="/">商品一覧へ戻る</Link>
      </Button>
    </div>
  );
}
