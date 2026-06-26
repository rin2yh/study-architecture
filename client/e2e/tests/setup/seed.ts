import { MEMBER_API_URL, ensureMember } from "./auth";

// product のマイグレーションはテーブル作成だけでシードを持たないため、E2E が前提とする商品を
// ここで用意する。store の loader は edge-proxy 経由で読むが、シード投入は product サービスへ
// 直接 (host 公開ポート) POST する。
const PRODUCT_API_URL = process.env.E2E_PRODUCT_API_URL ?? "http://localhost:8001";
// checkout は在庫予約を通る (ADR-[[202606262000]])。シード商品に十分な在庫を入れておかないと 409 で弾かれる。
const INVENTORY_API_URL = process.env.E2E_INVENTORY_API_URL ?? "http://localhost:8006";
const SEED_STOCK_QTY = 100_000;

interface SeedProduct {
  sku: string;
  name: string;
  priceCents: number;
}

interface ListedProduct extends SeedProduct {
  id: number;
}

export const SEED_PRODUCTS: readonly SeedProduct[] = [
  { sku: "E2E-001", name: "E2E テスト商品", priceCents: 12300 },
  { sku: "E2E-002", name: "E2E テスト商品(2)", priceCents: 4560 },
];

async function waitForHealthy(baseUrl: string, timeoutMs = 60_000): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  let lastError: unknown;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(`${baseUrl}/healthz`);
      if (res.ok) return;
      lastError = new Error(`healthz returned ${res.status}`);
    } catch (e) {
      lastError = e;
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`service not healthy at ${baseUrl}: ${String(lastError)}`);
}

async function seedProducts(): Promise<void> {
  const res = await fetch(`${PRODUCT_API_URL}/products`);
  if (!res.ok) throw new Error(`list products failed: ${res.status}`);
  const existing: SeedProduct[] = await res.json();
  const existingSkus = new Set(existing.map((p) => p.sku));

  // 複数回実行でも冪等にするため。
  for (const product of SEED_PRODUCTS) {
    if (existingSkus.has(product.sku)) continue;
    const created = await fetch(`${PRODUCT_API_URL}/products`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(product),
    });
    if (created.status !== 201) {
      throw new Error(`seed product ${product.sku} failed: ${created.status}`);
    }
  }
}

async function seedInventory(): Promise<void> {
  const res = await fetch(`${PRODUCT_API_URL}/products`);
  if (!res.ok) throw new Error(`list products failed: ${res.status}`);
  const products: ListedProduct[] = await res.json();
  const seedSkus = new Set(SEED_PRODUCTS.map((p) => p.sku));

  for (const product of products) {
    if (!seedSkus.has(product.sku)) continue;
    // 再実行で在庫が枯れない程度に上限まで補充する。append-only なので不足時のみ入庫する。
    const avail = await fetch(`${INVENTORY_API_URL}/availability/${product.id}`);
    if (!avail.ok) throw new Error(`get availability ${product.id} failed: ${avail.status}`);
    const { available }: { available: number } = await avail.json();
    if (available >= SEED_STOCK_QTY) continue;
    const stocked = await fetch(`${INVENTORY_API_URL}/stock-ins`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ productId: product.id, quantity: SEED_STOCK_QTY - available }),
    });
    if (stocked.status !== 201) {
      throw new Error(`seed stock for product ${product.id} failed: ${stocked.status}`);
    }
  }
}

export default async function globalSetup(): Promise<void> {
  await Promise.all([
    waitForHealthy(PRODUCT_API_URL),
    waitForHealthy(MEMBER_API_URL),
    waitForHealthy(INVENTORY_API_URL),
  ]);
  await seedProducts();
  await seedInventory();
  await ensureMember();
}
