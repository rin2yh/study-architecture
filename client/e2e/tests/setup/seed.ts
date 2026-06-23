import { MEMBER_API_URL, ensureMember } from "./auth";
import type { App } from "../stack/apps";

// product のマイグレーションはテーブル作成だけでシードを持たないため、E2E が前提とする商品を
// ここで用意する。store の loader は edge-proxy 経由で読むが、シード投入は product サービスへ
// 直接 (host 公開ポート) POST する。
const PRODUCT_API_URL = process.env.E2E_PRODUCT_API_URL ?? "http://localhost:8001";

interface SeedProduct {
  sku: string;
  name: string;
  priceCents: number;
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

// store はログインフローがあるため member も用意する。backoffice は閲覧のみで認証を要さない。
export async function seedForApp(app: App): Promise<void> {
  if (app === "store") {
    await Promise.all([waitForHealthy(PRODUCT_API_URL), waitForHealthy(MEMBER_API_URL)]);
    await seedProducts();
    await ensureMember();
    return;
  }
  await waitForHealthy(PRODUCT_API_URL);
  await seedProducts();
}
