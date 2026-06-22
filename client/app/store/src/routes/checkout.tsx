import { checkout } from "api/order";

import { currentMemberId } from "@/entities/session";
import { type CheckoutResult, parseItems } from "@/features/checkout";
import type { Route } from "./+types/checkout";

export { CheckoutPage as default } from "@/pages/checkout";

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
