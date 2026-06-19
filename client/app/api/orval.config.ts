import { defineConfig } from "orval";

// 全ドメインサービスの OpenAPI から fetch クライアント + zod スキーマを生成する共有パッケージ。
// 各 UI(apps/*) はここから必要なサービスだけ import する（重複排除）。
// baseURL はサーバ側ローダ/サーバ関数で env から注入する（mutator 参照）。
const service = (
  name: "product" | "order" | "payment" | "member" | "shipping",
  mutatorName: string,
) => ({
  [name]: {
    input: { target: `../../../server/${name}/api/openapi.yaml` },
    output: {
      mode: "tags-split" as const,
      client: "fetch" as const,
      httpClient: "fetch" as const,
      target: `src/gen/${name}/${name}.ts`,
      schemas: `src/gen/${name}/model`,
      indexFiles: true,
      override: {
        mutator: { path: "./src/mutator.ts", name: mutatorName },
      },
    },
  },
  [`${name}Zod`]: {
    input: { target: `../../../server/${name}/api/openapi.yaml` },
    output: {
      mode: "tags-split" as const,
      client: "zod" as const,
      target: `src/gen/${name}/${name}.zod.ts`,
      fileExtension: ".zod.ts",
    },
  },
});

export default defineConfig({
  ...service("product", "productFetch"),
  ...service("order", "orderFetch"),
  ...service("payment", "paymentFetch"),
  ...service("member", "memberFetch"),
  ...service("shipping", "shippingFetch"),
});
