import { listOrders, ListOrdersResponse } from "api/order";
import { redirect } from "react-router";
import type { Route } from "./+types/home";
import { currentMemberId } from "@/entities/session";
import { LogoutButton } from "@/features/auth";
import { OrderHistoryTable } from "./order-history-table";

export async function loader({ request }: Route.LoaderArgs) {
  const memberId = await currentMemberId(request);
  if (memberId === null) throw redirect("/login");

  const { data } = await listOrders({ headers: { "X-Member-Id": String(memberId) } });
  return { memberId, orders: ListOrdersResponse.parse(data) };
}

export default function Home({ loaderData }: Route.ComponentProps) {
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
