import type { Order } from "api/order";

interface OrderHistoryTableProps {
  orders: Order[];
}

export function OrderHistoryTable({ orders }: OrderHistoryTableProps) {
  return (
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
  );
}
