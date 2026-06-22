import type { Product } from "api/product";

interface ProductTableProps {
  products: Product[];
}

export function ProductTable({ products }: ProductTableProps) {
  return (
    <table className="mt-6 w-full border-collapse text-left text-sm">
      <thead>
        <tr className="border-b border-gray-300 text-gray-500">
          <th className="py-2 pr-4 font-medium">ID</th>
          <th className="py-2 pr-4 font-medium">SKU</th>
          <th className="py-2 pr-4 font-medium">商品名</th>
          <th className="py-2 pr-4 text-right font-medium">価格</th>
          <th className="py-2 font-medium">登録日時</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-gray-200">
        {products.map((p) => (
          <tr key={p.id}>
            <td className="py-3 pr-4 tabular-nums text-gray-500">{p.id}</td>
            <td className="py-3 pr-4 text-gray-500">{p.sku}</td>
            <td className="py-3 pr-4 font-medium">{p.name}</td>
            <td className="py-3 pr-4 text-right tabular-nums">
              ¥{(p.priceCents / 100).toLocaleString()}
            </td>
            <td className="py-3 tabular-nums text-gray-500">
              {new Date(p.createdAt).toLocaleString("ja-JP")}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
