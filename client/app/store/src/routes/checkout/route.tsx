import { useEffect } from "react";
import { useNavigation } from "react-router";

import { listAddresses, ListAddressesResponse } from "api/member";
import { checkout } from "api/order";
import { useCart } from "@/entities/cart";
import { currentMemberId, requireMemberId } from "@/features/auth";
import { CheckoutForm, parseItems, type CheckoutResult } from "@/features/checkout";
import { PageLoading } from "ui/page-loading";
import type { Route } from "./+types/route";
import { OrderConfirmed } from "./components/order-confirmed";
import { EmptyCheckout } from "./components/empty-checkout";
import { EmptyAddresses } from "./components/empty-addresses";

// ADR-[[202606170905]]
export async function loader({ request }: Route.LoaderArgs) {
  const memberId = await requireMemberId(request);
  const { data } = await listAddresses(memberId);
  return { addresses: ListAddressesResponse.parse(data) };
}

// ADR-[[202606170905]]
export async function action({ request }: Route.ActionArgs): Promise<CheckoutResult> {
  const form = await request.formData();
  const items = parseItems(form.get("items"));
  const paymentMethod = String(form.get("paymentMethod") ?? "");
  const shippingAddressId = Number(form.get("shippingAddressId"));

  if (items.length === 0) return { ok: false, error: "カートが空です。" };
  if (!paymentMethod) return { ok: false, error: "支払い方法を選択してください。" };
  // 宛先不明の出荷を作らないため、配送先未選択を確定前に弾く (ADR-[[202606261704]])。
  if (!Number.isInteger(shippingAddressId) || shippingAddressId <= 0) {
    return { ok: false, error: "配送先を選択してください。" };
  }

  const memberId = await currentMemberId(request);
  if (memberId === null) return { ok: false, error: "ログインが必要です。" };

  try {
    const res = await checkout({ memberId, shippingAddressId, paymentMethod, items });
    if (res.status !== 201) throw new Error(`checkout returned ${res.status}`);
    return { ok: true, order: res.data };
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : "確定に失敗しました。" };
  }
}

export default function Checkout({ loaderData, actionData }: Route.ComponentProps) {
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
  if (loaderData.addresses.length === 0) return <EmptyAddresses />;
  return (
    <CheckoutForm
      items={cart.items}
      addresses={loaderData.addresses}
      error={actionData?.error}
      submitting={navigation.state === "submitting"}
    />
  );
}
