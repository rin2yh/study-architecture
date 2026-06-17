import { listProducts, ListProductsResponse } from "@ec/api/product";
import type { Route } from "./+types/home";

export async function loader() {
  const { data } = await listProducts();
  return ListProductsResponse.parse(data);
}

export default function Home({ loaderData }: Route.ComponentProps) {
  const products = loaderData;
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

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  const message = error instanceof Error ? error.message : "unknown error";
  return (
    <div className="mx-auto max-w-4xl p-8" role="alert">
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
    <div className="mx-auto max-w-4xl p-8 text-gray-500" role="status" aria-live="polite">
      読み込み中…
    </div>
  );
}
