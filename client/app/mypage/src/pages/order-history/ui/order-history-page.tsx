import type { Order } from "api/order";

import { LogoutButton } from "@/features/auth";
import { OrderHistoryTable } from "./order-history-table";

export function OrderHistoryPage({
  loaderData,
}: {
  loaderData: { memberId: number; orders: Order[] };
}) {
  const { memberId, orders } = loaderData;
  return (
    <div className="mx-auto max-w-3xl p-8">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">注文履歴</h1>
        <div className="flex items-center gap-3 text-sm text-gray-500">
          <span>会員ID: {memberId}</span>
          <LogoutButton />
        </div>
      </div>
      <OrderHistoryTable orders={orders} />
      {orders.length === 0 && <p className="mt-6 text-gray-500">注文履歴がありません。</p>}
    </div>
  );
}
