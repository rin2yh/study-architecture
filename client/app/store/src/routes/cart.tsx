import { Link } from "react-router";

import { cartTotalCents } from "../cart";
import { yen } from "../money";
import { useCart } from "../use-cart";

export default function Cart() {
  const cart = useCart();

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
        <h1 className="text-3xl font-bold">カート</h1>
        <p className="mt-6 text-gray-500">カートは空です。</p>
        <Link to="/" className="mt-4 inline-block text-blue-600 underline">
          商品一覧へ
        </Link>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold">カート</h1>
      <ul className="mt-6 divide-y divide-gray-200">
        {cart.items.map((i) => (
          <li key={i.productId} className="flex items-center justify-between py-3">
            <div>
              <p className="font-medium">{i.name}</p>
              <p className="text-sm text-gray-500">{yen(i.priceCents)}</p>
            </div>
            <div className="flex items-center gap-3">
              <button
                type="button"
                aria-label="減らす"
                onClick={() => cart.setQty(i.productId, i.quantity - 1)}
                className="rounded border px-2"
              >
                −
              </button>
              <span className="w-6 text-center tabular-nums">{i.quantity}</span>
              <button
                type="button"
                aria-label="増やす"
                onClick={() => cart.setQty(i.productId, i.quantity + 1)}
                className="rounded border px-2"
              >
                ＋
              </button>
              <button
                type="button"
                onClick={() => cart.remove(i.productId)}
                className="ml-2 text-sm text-red-600 underline"
              >
                削除
              </button>
            </div>
          </li>
        ))}
      </ul>
      <div className="mt-6 flex items-center justify-between">
        <span className="text-lg font-bold">合計</span>
        <span className="text-lg font-bold tabular-nums">{yen(cartTotalCents(cart.items))}</span>
      </div>
      <Link to="/checkout" className="mt-6 inline-block rounded bg-blue-600 px-4 py-2 text-white">
        チェックアウトへ
      </Link>
    </div>
  );
}
