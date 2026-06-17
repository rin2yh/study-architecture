import { createFileRoute } from "@tanstack/react-router";

import { listProducts, ListProductsResponse } from "@ec/api/product";

export const Route = createFileRoute("/")({
  // ローダは SSR 時にサーバ側で実行される。mutator が env から service URL を注入する orval
  // クライアントで product を直接呼び、生成 zod で検証する。
  loader: async () => {
    const { data } = await listProducts();
    return ListProductsResponse.parse(data);
  },
  component: Home,
});

function Home() {
  const products = Route.useLoaderData();

  return (
    <div className="mx-auto max-w-2xl p-8">
      <h1 className="text-3xl font-bold">商品一覧</h1>
      <ul className="mt-6 divide-y divide-gray-200">
        {products.map((p) => (
          <li key={p.id} className="flex items-center justify-between py-3">
            <div>
              <p className="font-medium">{p.name}</p>
              <p className="text-sm text-gray-500">{p.sku}</p>
            </div>
            <span className="tabular-nums">¥{(p.priceCents / 100).toLocaleString()}</span>
          </li>
        ))}
      </ul>
      {products.length === 0 && <p className="mt-6 text-gray-500">商品がありません。</p>}
    </div>
  );
}
