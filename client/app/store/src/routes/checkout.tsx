import { useEffect } from "react";
import { Form, Link, useNavigation } from "react-router";
import { CheckCircle2 } from "lucide-react";

import { checkout, type Order } from "api/order";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { PageLoading } from "@/components/page-loading";
import { type CartItem, cartTotalCents, toCheckoutItems } from "../cart";
import { yen } from "../money";
import { currentMemberId } from "../session";
import { useCart } from "../use-cart";
import type { Route } from "./+types/checkout";

interface CheckoutItemInput {
  productId: number;
  quantity: number;
}

function parseItems(raw: FormDataEntryValue | null): CheckoutItemInput[] {
  try {
    const parsed: unknown = JSON.parse(String(raw ?? "[]"));
    return Array.isArray(parsed) ? (parsed as CheckoutItemInput[]) : [];
  } catch {
    return [];
  }
}

// ADR-[[202606170905]]
export async function action({ request }: Route.ActionArgs) {
  const form = await request.formData();
  const items = parseItems(form.get("items"));
  const paymentMethod = String(form.get("paymentMethod") ?? "");

  if (items.length === 0) return { ok: false as const, error: "カートが空です。" };
  if (!paymentMethod) return { ok: false as const, error: "支払い方法を選択してください。" };

  const memberId = await currentMemberId(request);
  if (memberId === null) return { ok: false as const, error: "ログインが必要です。" };

  try {
    const res = await checkout({ memberId, paymentMethod, items });
    if (res.status !== 201) throw new Error(`checkout returned ${res.status}`);
    return { ok: true as const, order: res.data };
  } catch (e) {
    return { ok: false as const, error: e instanceof Error ? e.message : "確定に失敗しました。" };
  }
}

function OrderConfirmed({ order }: { order: Order }) {
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

function EmptyCheckout() {
  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold">チェックアウト</h1>
      <p className="mt-6 text-muted-foreground">カートが空です。</p>
      <Button asChild variant="link" className="mt-4 px-0">
        <Link to="/">商品一覧へ</Link>
      </Button>
    </div>
  );
}

function CheckoutForm({
  items,
  error,
  submitting,
}: {
  items: CartItem[];
  error?: string;
  submitting: boolean;
}) {
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

export default function Checkout({ actionData }: Route.ComponentProps) {
  const cart = useCart();
  const navigation = useNavigation();

  const succeeded = actionData?.ok ?? false;
  const { clear } = cart;
  useEffect(() => {
    if (succeeded) clear();
  }, [succeeded, clear]);

  if (actionData?.ok) return <OrderConfirmed order={actionData.order} />;
  if (!cart.ready) return <PageLoading />;
  if (cart.items.length === 0) return <EmptyCheckout />;
  return (
    <CheckoutForm
      items={cart.items}
      error={actionData?.error}
      submitting={navigation.state === "submitting"}
    />
  );
}
