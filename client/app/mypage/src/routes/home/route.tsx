import { listOrders, ListOrdersResponse } from "api/order";
import { redirect } from "react-router";
import { Alert, AlertDescription, AlertTitle } from "ui/alert";
import { PageLoading } from "ui/page-loading";
import type { Route } from "./+types/route";
import { currentMemberId } from "@/entities/session";
import { LogoutButton } from "@/features/auth";
import { OrderHistoryTable } from "./components/order-history-table";

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
        <div className="flex items-center gap-3 text-sm text-muted-foreground">
          <span>会員ID: {memberId}</span>
          <LogoutButton />
        </div>
      </div>
      <OrderHistoryTable orders={orders} />
      {orders.length === 0 && <p className="mt-6 text-muted-foreground">注文履歴がありません。</p>}
    </div>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  const message = error instanceof Error ? error.message : "unknown error";
  return (
    <div className="mx-auto max-w-3xl p-8">
      <Alert variant="destructive">
        <AlertTitle>エラーが発生しました</AlertTitle>
        <AlertDescription>
          <p>注文履歴の取得に失敗しました。</p>
          <pre className="overflow-x-auto text-xs">{message}</pre>
        </AlertDescription>
      </Alert>
    </div>
  );
}

export function HydrateFallback() {
  return <PageLoading className="max-w-3xl" />;
}
