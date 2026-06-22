import type { Order } from "api/order";

export type CheckoutResult = { ok: false; error: string } | { ok: true; order: Order };
