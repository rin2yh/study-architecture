import { listOrders, ListOrdersResponse } from "@ec/api/order";
import type { Route } from "./+types/home";

export async function loader() {
  const { data } = await listOrders();
  return ListOrdersResponse.parse(data);
}

export default function Home({ loaderData }: Route.ComponentProps) {
  const orders = loaderData;
  return (
    <div className="mx-auto max-w-3xl p-8">
      <h1 className="text-3xl font-bold">注文履歴</h1>
      <table className="mt-6 w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-gray-300 text-left text-gray-500">
            <th className="py-2 pr-4 font-medium">注文ID</th>
            <th className="py-2 pr-4 font-medium">会員ID</th>
            <th className="py-2 pr-4 font-medium">ステータス</th>
            <th className="py-2 pr-4 text-right font-medium">合計</th>
            <th className="py-2 font-medium">注文日時</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200">
          {orders.map((o) => (
            <tr key={o.id}>
              <td className="py-3 pr-4 tabular-nums">{o.id}</td>
              <td className="py-3 pr-4 tabular-nums">{o.memberId}</td>
              <td className="py-3 pr-4">{o.status}</td>
              <td className="py-3 pr-4 text-right tabular-nums">
                ¥{(o.totalCents / 100).toLocaleString()}
              </td>
              <td className="py-3">{new Date(o.createdAt).toLocaleString("ja-JP")}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {orders.length === 0 && <p className="mt-6 text-gray-500">注文履歴がありません。</p>}
    </div>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  const message = error instanceof Error ? error.message : "unknown error";
  return (
    <div className="mx-auto max-w-3xl p-8" role="alert">
      <h1 className="text-3xl font-bold">エラーが発生しました</h1>
      <p className="mt-4 text-red-600">注文履歴の取得に失敗しました。</p>
      <pre className="mt-4 overflow-x-auto rounded bg-gray-100 p-3 text-xs text-gray-700">
        {message}
      </pre>
    </div>
  );
}

export function HydrateFallback() {
  return (
    <div className="mx-auto max-w-3xl p-8 text-gray-500" role="status" aria-live="polite">
      読み込み中…
    </div>
  );
}
