import type { Product } from "api/product";

import { ProductTable } from "./product-table";

export function ProductListPage({ loaderData }: { loaderData: Product[] }) {
  const products = loaderData;
  return (
    <div className="mx-auto max-w-4xl p-8">
      <h1 className="text-3xl font-bold">商品管理</h1>
      <p className="mt-2 text-sm text-gray-500">product サービスの ListProducts を一覧表示</p>
      <ProductTable products={products} />
      {products.length === 0 && <p className="mt-6 text-gray-500">商品がありません。</p>}
    </div>
  );
}
