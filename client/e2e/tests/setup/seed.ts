import { MEMBER_API_URL, ensureMember } from "./auth";
import type { App } from "../stack/apps";
import { appConfig } from "../stack/apps";
import { waitForOk } from "../stack/wait";

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

export async function seedForApp(app: App): Promise<void> {
  const waits = [waitForOk(`${PRODUCT_API_URL}/healthz`)];
  if (appConfig[app].needsMember) waits.push(waitForOk(`${MEMBER_API_URL}/healthz`));
  await Promise.all(waits);

  await seedProducts();
  if (appConfig[app].needsMember) await ensureMember();
}
