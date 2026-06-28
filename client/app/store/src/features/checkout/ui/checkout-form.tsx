import { Form } from "react-router";

import type { Address } from "api/member";
import { Alert, AlertDescription } from "ui/alert";
import { Button } from "ui/button";
import { Card, CardContent } from "ui/card";
import { Label } from "ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "ui/select";
import { Separator } from "ui/separator";
import { type CartItem, cartTotalCents } from "@/entities/cart";
import { yen } from "@/shared/lib/money";
import { toCheckoutItems } from "../model/checkout-items";
import { formatAddress } from "../model/format-address";

interface CheckoutFormProps {
  items: CartItem[];
  addresses: Address[];
  error?: string;
  submitting: boolean;
}

export function CheckoutForm({ items, addresses, error, submitting }: CheckoutFormProps) {
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

      {addresses.length === 0 ? (
        <Alert className="mt-8">
          <AlertDescription>
            配送先が登録されていません。住所を登録してから注文してください。
          </AlertDescription>
        </Alert>
      ) : (
        <Form method="post" className="mt-8 space-y-4">
          <input type="hidden" name="items" value={JSON.stringify(toCheckoutItems(items))} />
          <div className="space-y-2">
            <Label htmlFor="shippingAddressId">配送先</Label>
            <Select name="shippingAddressId" defaultValue={String(addresses[0].id)}>
              <SelectTrigger id="shippingAddressId" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {addresses.map((a) => (
                  <SelectItem key={a.id} value={String(a.id)}>
                    {formatAddress(a)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
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
      )}
    </div>
  );
}
