import { listOrders, ListOrdersResponse } from "api/order";
import { redirect } from "react-router";
import type { Route } from "./+types/home";
import { currentMemberId } from "@/entities/session";

export {
  OrderHistoryPage as default,
  OrderHistoryErrorBoundary as ErrorBoundary,
  OrderHistoryHydrateFallback as HydrateFallback,
} from "@/pages/order-history";

export async function loader({ request }: Route.LoaderArgs) {
  const memberId = await currentMemberId(request);
  if (memberId === null) throw redirect("/login");

  const { data } = await listOrders({ headers: { "X-Member-Id": String(memberId) } });
  return { memberId, orders: ListOrdersResponse.parse(data) };
}
