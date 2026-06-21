import { Link } from "react-router";

import { listProducts, ListProductsResponse } from "api/product";
import { yen } from "../money";
import { useCart } from "../use-cart";
import type { Route } from "./+types/home";

export async function loader() {
  const { data } = await listProducts();
  return ListProductsResponse.parse(data);
}

export default function Home({ loaderData }: Route.ComponentProps) {
  const products = loaderData;
  const cart = useCart();
  return (
    <div className="mx-auto max-w-2xl p-8">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">商品一覧</h1>
        <Link to="/cart" className="text-sm text-blue-600 underline">
          カート
        </Link>
      </div>
      <ul className="mt-6 divide-y divide-gray-200">
        {products.map((p) => (
          <li key={p.id} className="flex items-center justify-between py-3">
            <div>
              <p className="font-medium">{p.name}</p>
              <p className="text-sm text-gray-500">{p.sku}</p>
            </div>
            <div className="flex items-center gap-4">
              <span className="tabular-nums">{yen(p.priceCents)}</span>
              <button
                type="button"
                disabled={!cart.ready}
                onClick={() =>
                  cart.add({ productId: p.id, name: p.name, priceCents: p.priceCents })
                }
                className="rounded bg-blue-600 px-3 py-1 text-sm text-white disabled:opacity-50"
              >
                カートに入れる
              </button>
            </div>
          </li>
        ))}
      </ul>
      {products.length === 0 && <p className="mt-6 text-gray-500">商品がありません。</p>}
    </div>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  const message = error instanceof Error ? error.message : "unknown error";
  return (
    <div className="mx-auto max-w-2xl p-8" role="alert">
      <h1 className="text-3xl font-bold">エラーが発生しました</h1>
      <p className="mt-4 text-red-600">商品一覧の取得に失敗しました。</p>
      <pre className="mt-4 overflow-x-auto rounded bg-gray-100 p-3 text-xs text-gray-700">
        {message}
      </pre>
    </div>
  );
}

export function HydrateFallback() {
  return (
    <div className="mx-auto max-w-2xl p-8 text-gray-500" role="status" aria-live="polite">
      読み込み中…
    </div>
  );
}
