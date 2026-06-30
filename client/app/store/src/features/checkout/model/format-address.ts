import type { Address } from "api/member";

export function formatAddress(a: Address): string {
  return `${a.recipient} / 〒${a.postalCode} ${a.prefecture}${a.city} ${a.line1}`;
}
