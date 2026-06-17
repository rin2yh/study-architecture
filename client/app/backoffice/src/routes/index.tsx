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
    <div className="mx-auto max-w-4xl p-8 text-gray-500" role="status" aria-live="polite">
      読み込み中…
    </div>
  );
}

function ErrorView({ error }: ErrorComponentProps) {
  return (
    <div className="mx-auto max-w-4xl p-8" role="alert">
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
    <div className="mx-auto max-w-4xl p-8">
      <h1 className="text-3xl font-bold">商品管理</h1>
      <p className="mt-2 text-sm text-gray-500">product サービスの ListProducts を一覧表示</p>
      <table className="mt-6 w-full border-collapse text-left text-sm">
        <thead>
          <tr className="border-b border-gray-300 text-gray-500">
            <th className="py-2 pr-4 font-medium">ID</th>
            <th className="py-2 pr-4 font-medium">SKU</th>
            <th className="py-2 pr-4 font-medium">商品名</th>
            <th className="py-2 pr-4 text-right font-medium">価格</th>
            <th className="py-2 font-medium">登録日時</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200">
          {products.map((p) => (
            <tr key={p.id}>
              <td className="py-3 pr-4 tabular-nums text-gray-500">{p.id}</td>
              <td className="py-3 pr-4 text-gray-500">{p.sku}</td>
              <td className="py-3 pr-4 font-medium">{p.name}</td>
              <td className="py-3 pr-4 text-right tabular-nums">
                ¥{(p.priceCents / 100).toLocaleString()}
              </td>
              <td className="py-3 tabular-nums text-gray-500">
                {new Date(p.createdAt).toLocaleString("ja-JP")}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {products.length === 0 && <p className="mt-6 text-gray-500">商品がありません。</p>}
    </div>
  );
}
