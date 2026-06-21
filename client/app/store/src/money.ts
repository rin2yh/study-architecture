export function yen(cents: number): string {
  return `¥${(cents / 100).toLocaleString()}`;
}
