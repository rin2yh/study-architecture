import { listOrders } from "api/order";

// ADR-[[202606230930]]
export function listMyOrders(memberId: number) {
  return listOrders({ headers: { "X-Member-Id": String(memberId) } });
}
