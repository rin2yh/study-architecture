import { useEffect } from "react";
import { useNavigation } from "react-router";

import { checkout } from "api/order";
import { useCart } from "@/entities/cart";
import { currentMemberId } from "@/entities/session";
import { CheckoutForm, parseItems, type CheckoutResult } from "@/features/checkout";
import { PageLoading } from "@/shared/ui/page-loading";
import type { Route } from "./+types/checkout";
import { OrderConfirmed } from "./order-confirmed";
import { EmptyCheckout } from "./empty-checkout";

// ADR-[[202606170905]]
export async function action({ request }: Route.ActionArgs): Promise<CheckoutResult> {
  const form = await request.formData();
  const items = parseItems(form.get("items"));
  const paymentMethod = String(form.get("paymentMethod") ?? "");

  if (items.length === 0) return { ok: false, error: "カートが空です。" };
  if (!paymentMethod) return { ok: false, error: "支払い方法を選択してください。" };

  const memberId = await currentMemberId(request);
  if (memberId === null) return { ok: false, error: "ログインが必要です。" };

  try {
    const res = await checkout({ memberId, paymentMethod, items });
    if (res.status !== 201) throw new Error(`checkout returned ${res.status}`);
    return { ok: true, order: res.data };
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : "確定に失敗しました。" };
  }
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
