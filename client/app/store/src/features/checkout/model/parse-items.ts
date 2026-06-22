export interface CheckoutItemInput {
  productId: number;
  quantity: number;
}

export function parseItems(raw: FormDataEntryValue | null): CheckoutItemInput[] {
  try {
    const parsed: unknown = JSON.parse(String(raw ?? "[]"));
    return Array.isArray(parsed) ? (parsed as CheckoutItemInput[]) : [];
  } catch {
    return [];
  }
}
