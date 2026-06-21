import { useEffect } from "react";
import { Form, Link, useNavigation } from "react-router";

import { checkout } from "api/order";
import { cartTotalCents, toCheckoutItems } from "../cart";
import { yen } from "../money";
import { getCurrentMemberId } from "../session";
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

// カートはクライアント状態なので明細は hidden field で action に渡す ([[0006]])。
export async function action({ request }: Route.ActionArgs) {
  const form = await request.formData();
  const items = parseItems(form.get("items"));
  const paymentMethod = String(form.get("paymentMethod") ?? "");

  if (items.length === 0) return { ok: false as const, error: "カートが空です。" };
  if (!paymentMethod) return { ok: false as const, error: "支払い方法を選択してください。" };

  try {
    const { data } = await checkout({ memberId: getCurrentMemberId(), paymentMethod, items });
    // mutator は非 2xx で throw するので、ここに来た data は成功レスポンス (Order)。
    if (!("id" in data)) return { ok: false as const, error: "確定に失敗しました。" };
    return { ok: true as const, order: data };
  } catch (e) {
    return { ok: false as const, error: e instanceof Error ? e.message : "確定に失敗しました。" };
  }
}

export default function Checkout({ actionData }: Route.ComponentProps) {
  const cart = useCart();
  const navigation = useNavigation();
  const submitting = navigation.state === "submitting";

  const succeeded = actionData?.ok ?? false;
  const { clear } = cart;
  useEffect(() => {
    if (succeeded) clear();
  }, [succeeded, clear]);

  if (actionData?.ok) {
    const order = actionData.order;
    return (
      <div className="mx-auto max-w-2xl p-8">
        <h1 className="text-3xl font-bold">注文が確定しました</h1>
        <p className="mt-4">
          注文番号 <span className="font-mono">#{order.id}</span> / 合計{" "}
          <span className="tabular-nums">{yen(order.totalCents)}</span>
        </p>
        <ul className="mt-4 divide-y divide-gray-200">
          {(order.items ?? []).map((it) => (
            <li key={it.productId} className="flex justify-between py-2">
              <span>
                {it.productName} × {it.quantity}
              </span>
              <span className="tabular-nums">{yen(it.unitPriceCents * it.quantity)}</span>
            </li>
          ))}
        </ul>
        <Link to="/" className="mt-6 inline-block text-blue-600 underline">
          商品一覧へ戻る
        </Link>
      </div>
    );
  }

  if (!cart.ready) {
    return (
      <div className="mx-auto max-w-2xl p-8 text-gray-500" role="status" aria-live="polite">
        読み込み中…
      </div>
    );
  }

  if (cart.items.length === 0) {
    return (
      <div className="mx-auto max-w-2xl p-8">
        <h1 className="text-3xl font-bold">チェックアウト</h1>
        <p className="mt-6 text-gray-500">カートが空です。</p>
        <Link to="/" className="mt-4 inline-block text-blue-600 underline">
          商品一覧へ
        </Link>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold">チェックアウト</h1>
      <ul className="mt-6 divide-y divide-gray-200">
        {cart.items.map((i) => (
          <li key={i.productId} className="flex justify-between py-2">
            <span>
              {i.name} × {i.quantity}
            </span>
            <span className="tabular-nums">{yen(i.priceCents * i.quantity)}</span>
          </li>
        ))}
      </ul>
      <div className="mt-4 flex justify-between text-lg font-bold">
        <span>合計</span>
        <span className="tabular-nums">{yen(cartTotalCents(cart.items))}</span>
      </div>

      <Form method="post" className="mt-8 space-y-4">
        <input type="hidden" name="items" value={JSON.stringify(toCheckoutItems(cart.items))} />
        <label className="block">
          <span className="text-sm font-medium">支払い方法</span>
          <select
            name="paymentMethod"
            defaultValue="card"
            className="mt-1 block w-full rounded border p-2"
          >
            <option value="card">カード</option>
            <option value="bank_transfer">銀行振込</option>
            <option value="cod">代引き</option>
          </select>
        </label>
        {actionData?.error && (
          <p role="alert" className="text-red-600">
            {actionData.error}
          </p>
        )}
        <button
          type="submit"
          disabled={submitting}
          className="rounded bg-blue-600 px-4 py-2 text-white disabled:opacity-50"
        >
          {submitting ? "確定中…" : "注文を確定する"}
        </button>
      </Form>
    </div>
  );
}
