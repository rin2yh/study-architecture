import { createFileRoute } from "@tanstack/react-router";

import { listOrders, ListOrdersResponse } from "@ec/api/order";

export const Route = createFileRoute("/")({
  // ローダは SSR 時にサーバ側で実行される。mutator が env から service URL を注入する orval
  // クライアントで order を直接呼び、生成 zod で検証する。
  loader: async () => {
    const { data } = await listOrders();
    return ListOrdersResponse.parse(data);
  },
  component: Home,
});

function Home() {
  const orders = Route.useLoaderData();

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
