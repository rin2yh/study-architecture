// checkout の後段 (決済確定 → 配送手配 → 在庫確定) は UI を持たないため、E2E からは各サービスの
// host 公開ポートを直接叩いて観測する。決済確定はレプリカを増やすと固定ポートを共有できず
// ポートが振り替わるので env で差し替えられるようにしてある (ADR-[[202608160800]])。
const PAYMENT_API_URL = process.env.E2E_PAYMENT_API_URL ?? "http://localhost:8003";
const SHIPPING_API_URL = process.env.E2E_SHIPPING_API_URL ?? "http://localhost:8005";
const INVENTORY_API_URL = process.env.E2E_INVENTORY_API_URL ?? "http://localhost:8006";

interface Payment {
  id: number;
  orderId: number;
  status: string;
}

interface Shipment {
  id: number;
  orderId: number;
  status: string;
}

interface Reservation {
  id: number;
  productId: number;
  quantity: number;
  state: string;
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`);
  const body: T = await res.json();
  return body;
}

// 実運用では PSP からの通知や運用オペレーションが担う遷移。UI が無いので API を直接呼ぶ。
export async function settlePayment(orderId: number): Promise<void> {
  const payments = await getJSON<Payment[]>(`${PAYMENT_API_URL}/payments`);
  const payment = payments.find((p) => p.orderId === orderId);
  if (!payment) throw new Error(`no payment for order ${orderId}`);
  const res = await fetch(`${PAYMENT_API_URL}/payments/${payment.id}`, {
    method: "PUT",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ status: "settled" }),
  });
  if (!res.ok) throw new Error(`settle payment ${payment.id} failed: ${res.status}`);
}

export async function shipmentStatuses(orderId: number): Promise<string[]> {
  const shipments = await getJSON<Shipment[]>(`${SHIPPING_API_URL}/shipments`);
  return shipments.filter((s) => s.orderId === orderId).map((s) => s.status);
}

export async function reservationStates(orderId: number): Promise<string[]> {
  const reservations = await getJSON<Reservation[]>(`${INVENTORY_API_URL}/reservations/${orderId}`);
  return reservations.map((r) => r.state);
}
