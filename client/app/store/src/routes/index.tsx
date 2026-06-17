import { createFileRoute, type ErrorComponentProps } from "@tanstack/react-router";

import { listProducts, ListProductsResponse } from "@ec/api/product";

export const Route = createFileRoute("/")({
  // ローダは SSR 時にサーバ側で実行される。mutator が env から service URL を注入する orval
  // クライアントで product を直接呼び、生成 zod で検証する。
  loader: async () => {
    const { data } = await listProducts();
    return ListProductsResponse.parse(data);
  },
  component: Home,
  pendingComponent: Pending,
  errorComponent: ErrorView,
});

function Pending() {
  return (
    <div className="mx-auto max-w-2xl p-8 text-gray-500" role="status" aria-live="polite">
      読み込み中…
    </div>
  );
}

function ErrorView({ error }: ErrorComponentProps) {
  return (
    <div className="mx-auto max-w-2xl p-8" role="alert">
      <h1 className="text-3xl font-bold">エラーが発生しました</h1>
      <p className="mt-4 text-red-600">商品一覧の取得に失敗しました。</p>
      <pre className="mt-4 overflow-x-auto rounded bg-gray-100 p-3 text-xs text-gray-700">
        {error.message}
      </pre>
    </div>
  );
}

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
