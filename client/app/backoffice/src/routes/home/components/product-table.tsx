import type { Product } from "api/product";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "ui/table";

interface ProductTableProps {
  products: Product[];
}

export function ProductTable({ products }: ProductTableProps) {
  return (
    <Table className="mt-6">
      <TableHeader>
        <TableRow>
          <TableHead>ID</TableHead>
          <TableHead>SKU</TableHead>
          <TableHead>商品名</TableHead>
          <TableHead className="text-right">価格</TableHead>
          <TableHead>登録日時</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {products.map((p) => (
          <TableRow key={p.id}>
            <TableCell className="tabular-nums text-muted-foreground">{p.id}</TableCell>
            <TableCell className="text-muted-foreground">{p.sku}</TableCell>
            <TableCell className="font-medium">{p.name}</TableCell>
            <TableCell className="text-right tabular-nums">
              ¥{(p.priceCents / 100).toLocaleString()}
            </TableCell>
            <TableCell className="tabular-nums text-muted-foreground">
              {new Date(p.createdAt).toLocaleString("ja-JP")}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
