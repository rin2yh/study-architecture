import { listOrders } from "api/order";

const MEMBER_ID_HEADER = "X-Member-Id";

// ADR-[[202606230930]]
export function listMyOrders(memberId: number) {
  return listOrders(withMemberId(memberId));
}

function withMemberId(memberId: number): RequestInit {
  return { headers: { [MEMBER_ID_HEADER]: String(memberId) } };
}
