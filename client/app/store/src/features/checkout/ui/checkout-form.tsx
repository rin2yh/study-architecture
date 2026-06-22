import { Form } from "react-router";

import { Alert, AlertDescription } from "@/shared/ui/alert";
import { Button } from "@/shared/ui/button";
import { Card, CardContent } from "@/shared/ui/card";
import { Label } from "@/shared/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";
import { type CartItem, cartTotalCents, toCheckoutItems } from "@/entities/cart";
import { yen } from "@/shared/lib/money";

interface CheckoutFormProps {
  items: CartItem[];
  error?: string;
  submitting: boolean;
}

export function CheckoutForm({ items, error, submitting }: CheckoutFormProps) {
  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold">チェックアウト</h1>
      <Card className="mt-6">
        <CardContent className="space-y-3">
          {items.map((i) => (
            <div key={i.productId} className="flex justify-between">
              <span>
                {i.name} × {i.quantity}
              </span>
              <span className="tabular-nums">{yen(i.priceCents * i.quantity)}</span>
            </div>
          ))}
          <Separator />
          <div className="flex justify-between text-lg font-bold">
            <span>合計</span>
            <span className="tabular-nums">{yen(cartTotalCents(items))}</span>
          </div>
        </CardContent>
      </Card>

      <Form method="post" className="mt-8 space-y-4">
        <input type="hidden" name="items" value={JSON.stringify(toCheckoutItems(items))} />
        <div className="space-y-2">
          <Label htmlFor="paymentMethod">支払い方法</Label>
          <Select name="paymentMethod" defaultValue="card">
            <SelectTrigger id="paymentMethod" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="card">カード</SelectItem>
              <SelectItem value="bank_transfer">銀行振込</SelectItem>
              <SelectItem value="cod">代引き</SelectItem>
            </SelectContent>
          </Select>
        </div>
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
        <Button type="submit" disabled={submitting}>
          {submitting ? "確定中…" : "注文を確定する"}
        </Button>
      </Form>
    </div>
  );
}
