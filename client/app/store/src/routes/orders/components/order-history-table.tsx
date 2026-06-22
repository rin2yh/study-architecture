import type { Order } from "api/order";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "ui/table";

interface OrderHistoryTableProps {
  orders: Order[];
}

export function OrderHistoryTable({ orders }: OrderHistoryTableProps) {
  return (
    <Table className="mt-6">
      <TableHeader>
        <TableRow>
          <TableHead>注文ID</TableHead>
          <TableHead>会員ID</TableHead>
          <TableHead>ステータス</TableHead>
          <TableHead className="text-right">合計</TableHead>
          <TableHead>注文日時</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {orders.map((o) => (
          <TableRow key={o.id}>
            <TableCell className="tabular-nums">{o.id}</TableCell>
            <TableCell className="tabular-nums">{o.memberId}</TableCell>
            <TableCell>{o.status}</TableCell>
            <TableCell className="text-right tabular-nums">
              ¥{(o.totalCents / 100).toLocaleString()}
            </TableCell>
            <TableCell>{new Date(o.createdAt).toLocaleString("ja-JP")}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
