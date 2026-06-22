import { useEffect } from "react";
import { useNavigation } from "react-router";

import { useCart } from "@/entities/cart";
import { type CheckoutResult, CheckoutForm } from "@/features/checkout";
import { PageLoading } from "@/shared/ui/page-loading";
import { OrderConfirmed } from "./order-confirmed";
import { EmptyCheckout } from "./empty-checkout";

export function CheckoutPage({ actionData }: { actionData?: CheckoutResult }) {
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
